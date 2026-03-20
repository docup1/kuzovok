<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 * Бэкенд должен быть запущен на localhost:8080
 */

$backend_url = 'http://localhost:8080';
$request_uri = $_SERVER['REQUEST_URI'];

// Если запрос к статике или корень - отдаём index.html
if ($request_uri === '/' || $request_uri === '/index.html' || strpos($request_uri, '/static/') === 0) {
    // Проксируем статику с бэкенда
    $proxy_url = $backend_url . $request_uri;
} else {
    // Проксируем API запросы
    $proxy_url = $backend_url . $request_uri;
}

// Инициализируем cURL
$ch = curl_init($proxy_url);

// Копируем заголовки запроса (кроме Host)
$headers = [];
foreach (getallheaders() as $name => $value) {
    if ($name !== 'Host') {
        $headers[] = "$name: $value";
    }
}

curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $_SERVER['REQUEST_METHOD']);

// Передаём тело запроса для POST/PUT/PATCH
if ($_SERVER['REQUEST_METHOD'] === 'POST' || $_SERVER['REQUEST_METHOD'] === 'PUT' || $_SERVER['REQUEST_METHOD'] === 'PATCH') {
    $raw_input = file_get_contents('php://input');
    curl_setopt($ch, CURLOPT_POSTFIELDS, $raw_input);
}

// Выполняем запрос
$response = curl_exec($ch);
$http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
$content_type = curl_getinfo($ch, CURLINFO_CONTENT_TYPE);

if (curl_errno($ch)) {
    http_response_code(502);
    echo json_encode(['success' => false, 'message' => 'Backend unavailable: ' . curl_error($ch)]);
} else {
    http_response_code($http_code);
    
    // Устанавливаем Content-Type
    if ($content_type) {
        header("Content-Type: " . $content_type);
    }
    
    // Копируем заголовки ответа (кроме Transfer-Encoding)
    $header_size = curl_getinfo($ch, CURLINFO_HEADER_SIZE);
    $raw_headers = substr($response, 0, $header_size);
    preg_match_all('/^Set-Cookie:\s*(.+)$/m', $raw_headers, $matches, PREG_PATTERN_ORDER);
    foreach ($matches[1] as $cookie) {
        header("Set-Cookie: " . $cookie, false);
    }
    
    echo substr($response, $header_size);
}

curl_close($ch);
