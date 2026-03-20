<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 * Без cURL (использует file_get_contents)
 */

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

// API запросы - проксируем
if (strpos($request_uri, '/api/') === 0) {
    $proxy_url = $backend_url . $request_uri;
    
    $method = $_SERVER['REQUEST_METHOD'];
    $content = file_get_contents('php://input');
    
    $options = [
        'http' => [
            'method' => $method,
            'header' => "Content-Type: application/json\r\n",
            'timeout' => 30,
            'ignore_errors' => true
        ]
    ];
    
    if ($method === 'POST' || $method === 'PUT' || $method === 'PATCH') {
        $options['http']['content'] = $content;
    }
    
    $context = stream_context_create($options);
    $response = @file_get_contents($proxy_url, false, $context);
    
    if ($response === false) {
        http_response_code(502);
        header("Content-Type: application/json");
        echo json_encode(['success' => false, 'message' => 'Backend unavailable']);
    } else {
        header("Content-Type: application/json");
        echo $response;
    }
    exit;
}

// Статика
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

http_response_code(404);
echo "Not found: " . htmlspecialchars($request_uri);
