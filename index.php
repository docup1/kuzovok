<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 */

error_log("=== Kuzovok started ===");

$backend_url = 'http://127.0.0.1:8080';
$base_dir = __DIR__;
$static_dir = $base_dir . '/static';

// Получаем URI
$request_uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$base_path = '/~s409784/kuzovok';

if (strpos($request_uri, $base_path) === 0) {
    $request_uri = substr($request_uri, strlen($base_path));
}
if ($request_uri === '' || $request_uri === '/') {
    $request_uri = '/';
}

error_log("Request URI: $request_uri");

// API запросы - проксируем
if (strpos($request_uri, '/api/') === 0) {
    $proxy_url = $backend_url . $request_uri;
    error_log("Proxying API: $proxy_url");
    
    $ch = curl_init($proxy_url);
    if ($ch === false) {
        error_log("curl_init failed");
        http_response_code(500);
        echo json_encode(['error' => 'curl init failed']);
        exit;
    }

    $headers = [];
    if (function_exists('getallheaders')) {
        foreach (getallheaders() as $name => $value) {
            if ($name !== 'Host' && $name !== 'Connection' && $name !== 'Content-Length') {
                $headers[] = "$name: $value";
            }
        }
    }
    error_log("Headers: " . print_r($headers, true));

    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_TIMEOUT, 30);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $_SERVER['REQUEST_METHOD']);

    if ($_SERVER['REQUEST_METHOD'] === 'POST' || $_SERVER['REQUEST_METHOD'] === 'PUT' || $_SERVER['REQUEST_METHOD'] === 'PATCH') {
        $raw_input = file_get_contents('php://input');
        error_log("POST data: $raw_input");
        curl_setopt($ch, CURLOPT_POSTFIELDS, $raw_input);
    }

    $response = curl_exec($ch);
    $error = curl_error($ch);
    $http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    
    error_log("Response code: $http_code");
    error_log("Response: " . substr($response, 0, 200));
    if ($error) {
        error_log("cURL error: $error");
    }

    if ($error || !$response) {
        http_response_code(502);
        header("Content-Type: application/json");
        echo json_encode(['success' => false, 'message' => 'Backend: ' . $error]);
    } else {
        http_response_code($http_code);
        header("Content-Type: application/json");
        echo $response;
    }
    curl_close($ch);
    exit;
}

// Статика
if ($request_uri === '/' || $request_uri === '/index.html') {
    $file_path = $static_dir . '/index.html';
    error_log("Serving static: $file_path");
} else {
    $file_path = $static_dir . $request_uri;
    error_log("Serving file: $file_path");
}

if (file_exists($file_path)) {
    $mime_type = mime_content_type($file_path);
    header("Content-Type: " . $mime_type);
    readfile($file_path);
    exit;
}

error_log("404: $request_uri");
http_response_code(404);
echo "Not found: " . htmlspecialchars($request_uri);
