![图片](https://github.com/zwjzxh520/codeDemo/blob/main/resource/mp.png?raw=true)

> 原文地址 [mp.weixin.qq.com](https://mp.weixin.qq.com/s/wMJ6HAgiTvsd5fqD7gXRcw)

最近遇到一个需求：在一个已知的文件头部插入一部分数据。  

如果是在一个文件的尾部插入数据，只需要 **file_put_contents($filename, $data, FILE_APPEND)** 即可。  

由于源文件已经保存在了磁盘上，且它的数据大小不定，如果将原文件直接全部读出，再附加到想添加的数据之后，效率肯定不是最好的，因此想利用零拷贝 (zero-copy) 技术实现。  

零拷贝技术主要用于减少磁盘操作 (I/O) 时的 CPU 时间，在一定程度上，是一种效率比较高效的 I/O 处理方式。具体信息搜索一下，即能看到大量的资料。  

PHP 中的零拷贝实现，主要是利用 Linux 系统中自带的 sendfile 函数，通过阅读源码可知：

涉及到的函数有 **stream_copy_to_stream** ，其内部函数调用的是 **php_stream_copy_to_stream_ex** ，在查看 **file_put_contents** 时，发现当第一个参数 $filename 是一个 resource 时，同样是调用了 **php_stream_copy_to_stream_ex**。这一点，在 PHP 的官方手册中也有说明，详见：https://www.php.net/manual/zh/function.file-put-contents.php#refsect1-function.file-put-contents-parameters

源码文件：ext/standard/file.c

```
/* {{{ Write/Create a file with contents data and return the number of bytes written */
PHP_FUNCTION(file_put_contents)
{
  php_stream *stream;
  char *filename;
  size_t filename_len;
  ......
  switch (Z_TYPE_P(data)) {
    case IS_RESOURCE: {
      size_t len;
      if (php_stream_copy_to_stream_ex(srcstream, stream, PHP_STREAM_COPY_ALL, &len) != SUCCESS) {
        numbytes = -1;
      } else {
        if (len > ZEND_LONG_MAX) {
          php_error_docref(NULL, E_WARNING, "content truncated from %zu to " ZEND_LONG_FMT " bytes", len, ZEND_LONG_MAX);
          len = ZEND_LONG_MAX;
        }
        numbytes = len;
      }
      break;
    }
```

经过一番资料搜索之后，总结出了以下代码  

```
/**
 * 在文件头部插入数据
 * @Author Code Demo 大全
 */
function filePrepend($string, $origin_filename)
{
    $context = stream_context_create();
    $orig_file = fopen($origin_filename, 'r', 1, $context);
    $temp_filename = tempnam(sys_get_temp_dir(), 'php_prepend_');
    file_put_contents($temp_filename, $string);
    file_put_contents($temp_filename, $orig_file, FILE_APPEND);
    fclose($orig_file);
    unlink($origin_filename);
    rename($temp_filename, $origin_filename);
}
```

利用 xhprof 进行测试，检查性能如何。

测试的 php 版本为 8.0.13。运行于 docker 容器中，目标文件大小：229,743B

对比代码，将源文件内容读取后再附加要添加的数据，然后再写入源文件：

```
function filePrependCompare($string, $origin_filename)
{
    $content = file_get_contents($origin_filename);
    file_put_contents($origin_filename, $string.$content);
}
```

测试代码：  

```
$i = 0;
while($i < 100) {
    $i++;
    filePrepend('如果 data 指定为 stream 资源，这里 stream 中所保存的缓存数据将被写入到指定文件中，这种用法就相似于使用 stream_copy_to_stream() 函数。'.PHP_EOL, $src_file);
}
```

```
$i = 0;
while($i < 100) {
    $i++;
    filePrependCompare('如果 data 指定为 stream 资源，这里 stream 中所保存的缓存数据将被写入到指定文件中，这种用法就相似于使用 stream_copy_to_stream() 函数。'.PHP_EOL, $src_file);
}
```

测试结果：

<table><tbody><tr><th>指标<br></th><th>filePrepend</th><th>filePrependCompare</th></tr><tr><td width="171" valign="top"><p>Incl.Wall Time(microsec)</p></td><td width="171" valign="top">2,554,825</td><td width="227" valign="top">1,415,920</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.Wall Time</p><p>(microsec)</p></td><td valign="top" colspan="1" rowspan="1" width="171">2,554,825</td><td valign="top" colspan="1" rowspan="1" width="227">1,415,920</td></tr><tr><td width="171" valign="top">Incl.MemUse(bytes)‍</td><td width="171" valign="top">18,864</td><td width="247" valign="top">24,064,520</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.MemUse(bytes)</p></td><td valign="top" colspan="1" rowspan="1" width="171">18,864</td><td valign="top" colspan="1" rowspan="1" width="247">24,064,520</td></tr></tbody></table>

通过测试发现一个神奇的现象，看似性能更好的办法，占用的 CPU 时间反而更长。不过内存消耗是要少得更多。

这个看似不合理的地方，是怎么回事？  

其实仔细检查一下 **filePrepend** 方法，发现在数据写完之后 ，还有 unlink  和 rename 的调用，这明显也是需要消耗时间的。咱们先注释掉这两行代码试试看看。

```
/**
 * 在文件头部插入数据
 * @Author Code Demo 大全
 */
function filePrepend($string, $origin_filename)
{
    $context = stream_context_create();
    $orig_file = fopen($origin_filename, 'r', 1, $context);
    $temp_filename = tempnam(sys_get_temp_dir(), 'php_prepend_');
    file_put_contents($temp_filename, $string);
    file_put_contents($temp_filename, $orig_file, FILE_APPEND);
    fclose($orig_file);
    // unlink($origin_filename);
    // rename($temp_filename, $origin_filename);
}
```

测试结果：

<table><tbody><tr><th>指标<br></th><th>filePrepend</th><th>filePrependCompare</th></tr><tr><td width="171" valign="top"><p>Incl.Wall Time(microsec)</p></td><td width="171" valign="top">186,616</td><td width="227" valign="top">1,195,395</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.Wall Time</p><p>(microsec)</p></td><td valign="top" colspan="1" rowspan="1" width="171">186,616</td><td valign="top" colspan="1" rowspan="1" width="227">1,195,395</td></tr><tr><td width="171" valign="top">Incl.MemUse(bytes)</td><td width="171" valign="top">18,920</td><td width="247" valign="top">24,064,520</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.MemUse(bytes)</p></td><td valign="top" colspan="1" rowspan="1" width="171">18,920</td><td valign="top" colspan="1" rowspan="1" width="247">24,064,520</td></tr></tbody></table>

可以看到，去掉之后，CPU 大幅下降了。  

性能的要求达到了，但是去掉那两行后，功能要求是无法实现的，该如何处理？

在 PHP 的官方手册中，对 rename 的说明是：

> rename(string $from, string $to, ?resource $context = null): bool 
> 
> 尝试把 from 重命名为 to，必要时会在不同目录间移动。 如果重命名文件时 to 已经存在，将会覆盖掉它。 如果重‍命名文件夹时 to 已经存在，本函数将导致一个警告。

那我们完全只需要注释掉 unlink 函数，保留 rename 函数即可。新的代码如下：  

```
/**
 * 在文件头部插入数据
 * @Author Code Demo 大全
 */
function filePrepend($string, $origin_filename)
{
    $context = stream_context_create();
    $orig_file = fopen($origin_filename, 'r', 1, $context);
    $temp_filename = tempnam(sys_get_temp_dir(), 'php_prepend_');
    file_put_contents($temp_filename, $string);
    file_put_contents($temp_filename, $orig_file, FILE_APPEND);
    fclose($orig_file);
    //unlink($origin_filename);
    rename($temp_filename, $origin_filename);
}
```

测试结果：

<table><tbody><tr><th>指标<br></th><th>filePrepend</th><th>filePrependCompare</th></tr><tr><td width="171" valign="top"><p>Incl.Wall Time(microsec)</p></td><td width="171" valign="top">1,754,703</td><td width="227" valign="top">1,153,985</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.Wall Time</p><p>(microsec)</p></td><td valign="top" colspan="1" rowspan="1" width="171">1,754,703</td><td valign="top" colspan="1" rowspan="1" width="227">1,153,985</td></tr><tr><td width="171" valign="top">Incl.MemUse(bytes)</td><td width="171" valign="top">18,864</td><td width="247" valign="top">24,064,520</td></tr><tr><td valign="top" colspan="1" rowspan="1"><p>Excl.MemUse(bytes)</p></td><td valign="top" colspan="1" rowspan="1" width="171">18,864</td><td valign="top" colspan="1" rowspan="1" width="247">24,064,520</td></tr></tbody></table>

可以看到，CPU 消耗已经不像最开始那样的恐怖了。

至于为什么没有提供像 FILE_APPEND 类似的 FILE_PREPEND 标记，个人猜测是磁盘块方面的原因。

在 Laravel 的 github issue 中，也看到了类似的讨论  

> Additionally, from performance standpoint, these methods get all content from the original file and only then append/prepend to it. If the original file is very large, you get the idea. My question is: why streams weren't used instead? Streams could do these tasks much faster and cheaper.
> 
> https://github.com/laravel/framework/issues/11041

而在 stackoverflow 中，也看到了相关的解释

> I don't imagine that many filesystems would have a machanism for chaining partial blocks, but even if they do it would result in huge inefficiencies. You'd end up with a file consisting of mostly empty blocks, and you have to have to read and write the entire file to defragment it.
> 
> https://stackoverflow.com/a/5560246

总结：  

*   如果文件较小，不会导致内存溢出的问题，则可以直接使用 file_get_contents 将所有内容读取后，再附加数据，重新写入原文件即可。
    
*   如果文件较大，建议使用流的方式，cpu 消耗没有增长的太厉害，同时内存要小得多。不过要注意的是，会产生很多的临时文件，需要注意清理。  
    

您还有更好的方案吗？欢迎讨论。
