# 使用 signature_pad.js 对 pdf 进行签名

![图片](https://github.com/zwjzxh520/codeDemo/blob/main/resource/mp.png?raw=true)

> 原文地址 [Code Demo 大全](https://mp.weixin.qq.com/s/FQmLRc9VdaBaF6cz83-CjQ)

前端

html

前端代码只需要一个 canvas 元素就行了  

```
<canvas id="signatureCanvas"></canvas>
<div>
    <button id="clearButton">Clear</button>
    <button id="saveButton">Save</button>
</div>
```

js

js 代码主要使用 signature_pad.js，开源地址: 

https://github.com/szimek/signature_pad

引入 js：

```
<script src="https://cdnjs.cloudflare.com/ajax/libs/signature_pad/4.1.6/signature_pad.umd.min.js"
            integrity="sha512-EfX4vFXXWtDM8PcSpNZK3oVNpU50itrpemKPX6/KJTZnT/wM81S5HGzHs+G9lqBBjemL4GYoWVCjdhGP8qTHdA=="
            crossorigin="anonymous" referrerpolicy="no-referrer"></script>
```

实现手写功能：

```
(function () {
    const canvas = document.getElementById("signatureCanvas");
    const signaturePad = new SignaturePad(canvas, {penColor: 'rgb(0, 0, 0)'});
    scaleCanvas();

    const clearButton = document.getElementById("clearButton");

    clearButton.addEventListener("click", clearCanvas);

    function clearCanvas() {
        signaturePad.clear()
    }

    // 适应各种DPI的设备，否则可能会出现线条没在指尖下的情况。
    function scaleCanvas() {
        const ratio = Math.max(window.devicePixelRatio || 1, 1);
        canvas.width = canvas.offsetWidth * ratio;
        canvas.height = canvas.offsetHeight * ratio;
        canvas.getContext("2d").scale(ratio, ratio);
        signaturePad.clear(); // otherwise isEmpty() might return incorrect value
    }

    $(window).on("resize", scaleCanvas)


    const saveButton = document.getElementById("saveButton");

    saveButton.addEventListener("click", saveCanvas);

    function saveCanvas() {
        // 签名为空的判断
        if (signaturePad.isEmpty()) {
            alert("不能为空")
        }
        // signaturePad.jSignature('getData');
        var imgStr = signaturePad.toDataURL('image/png');
        //获取到image的base64 可以把这个传到后台解析成图片
        //imgStr = data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAfgAAAL2CAYAAA......
        //去掉data:image/png;base64,我们只要后面的部分iVBORw0KGgoAAAANSUhEUgAAAfgAAAL2CAYAAA......
        imgStr = imgStr.substring(22, imgStr.length);
        console.log(imgStr)
    }
})()
```

其中的 scaleCanvas() 至关重要，没有此代码，则可能会出现线条没有在指尖下的情况。

PHP

找到签名嵌入的位置

需要依赖一个外部工具 pdftotext，包名是 poppler-utils

```
sudo yum install poppler-utils
```

导出为 html 格式，并带上文字的大小及相对于页面的位置信息  

```
pdftotext -htmlmeta -bbox test.PDF test.html
```

生成的 test.html 有如下信息：  

```
<body>
<doc>
  <page width="609.000000" height="791.000000">
    <word xMin="341.600000" yMin="11.940000" xMax="418.800000" yMax="30.440000">Delivery</word>
    <word xMin="424.410000" yMin="11.940000" xMax="468.810000" yMax="30.440000">Note</word>
    <word xMin="266.500000" yMin="43.252000" xMax="288.445000" yMax="53.427000">DN#</word>
    <word xMin="266.500000" yMin="57.402000" xMax="283.649000" yMax="67.577000">Old</word>
  </page>
</doc>
</body>
```

通过 xMin，yMin，xMax，yMax 这两个坐标点，就能定位到文字块的位置及大小了。

签名域名，一般情况下，都是多个下划线组成的。  

将手写签名放到指定的位置

先安装依赖库  

```
composer require setasign/fpdf setasign/fpdi
```

全部代码

```
<?php
function attachSignature($pdf_file, $png)
{
    $position = searchPDF($pdf_file, '____');
    
    $pdf = new \setasign\Fpdi\Fpdi();
    $pageCount = $pdf->setSourceFile($pdf_file);

    if ($pageCount == 0) {
        throw new Exception('Read empty file');
    }
    $ratio = 2.83;

    for ($i = 1; $i <= $pageCount; $i++) {
        $tplIdx = $pdf->importPage($i);
        $pdf->addPage();
        $pdf->useTemplate($tplIdx, 0, 0);
        // 将图片插入到指定位置
        foreach ($position as $item) {
            if ($item['page']['pageIndex'] == $i) {
                // $item['yMin'] - 30 可以根据图片大小调整，将图片覆盖到指定签名区域上
                // 60, 20 则是覆盖后的图片大小
                $pdf->Image($png, $item['xMin'] / $ratio, ($item['yMin'] - 30) / $ratio, 60, 20, 'PNG');
            }
        }
    }

    $pageSize = $pdf->getImportedPageSize($tplIdx);
    if (!$pageSize) {
        throw new Exception('Page number reads as null value.');
    }
    // $position 中 page 长 宽 与 $pageSize 的长宽 比值

    $pdf->Output('F', $pdf_file);
}

/**
 * 在 pdf 中搜索特定字符及位置信息。需要安装 pdftotext 工具。
 * sudo yum install poppler-utils
 * 通过以下命令获取文本及位置信息
 * pdftotext -htmlmeta -bbox DN0080036162.PDF
 * @param string $pdf
 * @param string $text
 * @return array
 * @throws Exception
 */
function searchPDF($pdf, $text): array
{
    // 创建一个新的进程
    // pdftotext -htmlmeta -bbox DN0080036162.PDF
    $output_html = $pdf . '.html';
    // 创建一个新的进程
    $process = new \Symfony\Component\Process\Process(['pdftotext', '-htmlmeta', '-bbox', $pdf, $output_html]);

// 启动进程并等待其完成
    $process->run();

    if (!$process->isSuccessful()) {
        throw new Exception($process->getErrorOutput());
    }
    $result = [];
    $data = simplexml_load_file($output_html);
    if (!$data || !isset($data->body->doc->page)) {
        throw new Exception('Error PDF file. No page information was parsed.');
    }
    $page_index = 0;
    foreach ($data->body->doc->page as $page) {
        $page_index++;
        $page_attr = [
            'pageIndex' => $page_index,
            'width'     => (string)$page->attributes()['width'],
            'height'    => (string)$page->attributes()['height'],
        ];
        foreach ($page->children() as $word) {
            if (strpos($word, $text) !== false) {
                $attributes = $word->attributes();
                $result[] = [
                    'page' => $page_attr,
                    'xMin' => (string)$attributes['xMin'],
                    'yMin' => (string)$attributes['yMin'],
                    'xMax' => (string)$attributes['xMax'],
                    'yMax' => (string)$attributes['yMax'],
                    'word' => (string)$word,
                ];
            }
        }
    }
    return $result;
}
```

祝各位顺利！