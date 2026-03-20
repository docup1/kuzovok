<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 * Бэкенд должен быть запущен на localhost:8080
 */

$backend_url = 'http://127.0.0.1:8080';
$base_dir = __DIR__;
$static_dir = $base_dir . '/static';

// Получаем URI
$request_uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);

// Логирование
error_log("Kuzovok proxy: $request_uri");

// API запросы - проксируем
if (strpos($request_uri, '/api/') === 0) {
    $proxy_url = $backend_url . $request_uri;
    $ch = curl_init($proxy_url);

    $headers = [];
    foreach (getallheaders() as $name => $value) {
        if ($name !== 'Host' && $name !== 'Connection') {
            $headers[] = "$name: $value";
        }
    }

    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_TIMEOUT, 30);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $_SERVER['REQUEST_METHOD']);

    if ($_SERVER['REQUEST_METHOD'] === 'POST' || $_SERVER['REQUEST_METHOD'] === 'PUT' || $_SERVER['REQUEST_METHOD'] === 'PATCH') {
        $raw_input = file_get_contents('php://input');
        curl_setopt($ch, CURLOPT_POSTFIELDS, $raw_input);
    }

    $response = curl_exec($ch);
    $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $content_type = curl_getinfo($ch, CURLINFO_CONTENT_TYPE);

    if (curl_errno($ch)) {
        http_response_code(502);
        header("Content-Type: application/json");
        echo json_encode(['success' => false, 'message' => 'Backend: ' . curl_error($ch)]);
    } else {
        http_response_code($http_code);
        if ($content_type) {
            header("Content-Type: " . $content_type);
        }
        echo $response;
    }
    curl_close($ch);
    exit;
}

// Статика из папки static/
if ($request_uri === '/' || $request_uri === '/index.html') {
    $file_path = $static_dir . '/index.html';
} else {
    $file_path = $static_dir . $request_uri;
}

if (file_exists($file_path)) {
    $mime_type = mime_content_type($file_path);
    header("Content-Type: " . $mime_type);
    readfile($file_path);
    exit;
}

// 404
http_response_code(404);
echo "Not found: " . htmlspecialchars($request_uri);
