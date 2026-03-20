<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 * Бэкенд должен быть запущен на localhost:8080
 */

$backend_url = 'http://127.0.0.1:8080';
$request_uri = $_SERVER['REQUEST_URI'];

// Логирование для отладки
error_log("Proxy request: $request_uri");

// Если запрос к API - проксируем на бэкенд
if (strpos($request_uri, '/api/') === 0) {
    $proxy_url = $backend_url . $request_uri;
} else {
    // Для статики и корня - читаем файл из static/
    if ($request_uri === '/' || $request_uri === '/index.html') {
        $file_path = __DIR__ . '/static/index.html';
    } else {
        $file_path = __DIR__ . '/static' . $request_uri;
    }
    
    if (file_exists($file_path)) {
        $mime_type = mime_content_type($file_path);
        header("Content-Type: " . $mime_type);
        readfile($file_path);
        exit;
    } else {
        http_response_code(404);
        echo "File not found: " . htmlspecialchars($request_uri);
        exit;
    }
}

// Инициализируем cURL для API запросов
$ch = curl_init($proxy_url);

// Копируем заголовки запроса
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

// Передаём тело запроса для POST/PUT/PATCH
if ($_SERVER['REQUEST_METHOD'] === 'POST' || $_SERVER['REQUEST_METHOD'] === 'PUT' || $_SERVER['REQUEST_METHOD'] === 'PATCH') {
    $raw_input = file_get_contents('php://input');
    curl_setopt($ch, CURLOPT_POSTFIELDS, $raw_input);
    error_log("POST data: " . substr($raw_input, 0, 200));
}

// Выполняем запрос
$response = curl_exec($ch);
$http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
$content_type = curl_getinfo($ch, CURLINFO_CONTENT_TYPE);

if (curl_errno($ch)) {
    error_log("cURL error: " . curl_error($ch));
    http_response_code(502);
    header("Content-Type: application/json");
    echo json_encode(['success' => false, 'message' => 'Backend unavailable: ' . curl_error($ch)]);
} else {
    http_response_code($http_code);
    
    // Устанавливаем Content-Type
    if ($content_type) {
        header("Content-Type: " . $content_type);
    }
    
    // Копируем заголовки Set-Cookie
    $header_size = curl_getinfo($ch, CURLINFO_HEADER_SIZE);
    $raw_headers = substr($response, 0, $header_size);
    preg_match_all('/^Set-Cookie:\s*(.+)$/m', $raw_headers, $matches, PREG_PATTERN_ORDER);
    foreach ($matches[1] as $cookie) {
        header("Set-Cookie: " . $cookie, false);
    }
    
    error_log("Response: " . substr($response, $header_size, 200));
    echo substr($response, $header_size);
}

curl_close($ch);
