<?php
// Проверка структуры файлов
$base = __DIR__;
echo "<pre>";
echo "Base dir: $base\n\n";

$files = ['index.php', '.htaccess', 'kusovok', 'kusovok.db', 'static/index.html'];
foreach ($files as $f) {
    $path = $base . '/' . $f;
    $exists = file_exists($path) ? '✓ EXISTS' : '✗ MISSING';
    $isDir = is_dir($path) ? '(dir)' : '(file)';
    echo "$f: $exists $isDir\n";
}
echo "</pre>";
?>
