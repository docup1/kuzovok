<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 */

$config_path = __DIR__ . '/config.json';
$config = file_exists($config_path) ? json_decode(file_get_contents($config_path), true) : [];

$backend_url = $config['proxy']['backend_url'] ?? getenv('KUSOVOK_BACKEND_URL') ?: 'http://127.0.0.1:8080';
$proxy_driver = strtolower($config['proxy']['driver'] ?? getenv('KUSOVOK_PROXY_DRIVER') ?: 'auto');
$base_dir = __DIR__;
$static_dir = $base_dir . '/static';
$image_dir = $base_dir . '/img';

$request_uri = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
$script_dir = str_replace('\\', '/', dirname($_SERVER['SCRIPT_NAME'] ?? ''));
$base_path = $script_dir === '/' || $script_dir === '.' ? '' : rtrim($script_dir, '/');
$cookie_path = $base_path === '' ? '/' : $base_path . '/';
$is_https = (
    (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
    || (isset($_SERVER['SERVER_PORT']) && (string)$_SERVER['SERVER_PORT'] === '443')
    || (isset($_SERVER['HTTP_X_FORWARDED_PROTO']) && $_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https')
);

if ($base_path !== '' && strpos($request_uri, $base_path) === 0) {
    $request_uri = substr($request_uri, strlen($base_path));
}
if ($request_uri === '' || $request_uri === '/') {
    $request_uri = '/';
}

if (strpos($request_uri, '/api/') === 0) {
    proxy_api_request($backend_url . $request_uri, $cookie_path, $is_https, $proxy_driver);
    exit;
}

if (strpos($request_uri, '/img/') === 0) {
    serve_local_file($image_dir, $request_uri, true);
    exit;
}

if ($request_uri === '/admin' || $request_uri === '/admin/' || $request_uri === '/admin.html') {
    $file_path = $static_dir . '/admin.html';
} elseif ($request_uri === '/profile' || $request_uri === '/profile/' || $request_uri === '/profile.html') {
    $file_path = $static_dir . '/profile.html';
} elseif (strpos($request_uri, '/user/') === 0) {
    $file_path = $static_dir . '/user.html';
} elseif ($request_uri === '/' || $request_uri === '/index.html') {
    $file_path = $static_dir . '/index.html';
} else {
    $file_path = $static_dir . $request_uri;
}

if (is_file($file_path)) {
    header('Content-Type: ' . (mime_content_type($file_path) ?: 'application/octet-stream'));
    readfile($file_path);
    exit;
}

http_response_code(404);
echo 'Not found: ' . htmlspecialchars($request_uri, ENT_QUOTES, 'UTF-8');

function proxy_api_request(string $proxy_url, string $cookie_path, bool $is_https, string $proxy_driver): void
{
    $force_stream = $proxy_driver === 'stream';
    $force_curl = $proxy_driver === 'curl';

    if ($force_stream || (!$force_curl && !function_exists('curl_init'))) {
        proxy_api_request_via_stream($proxy_url, $cookie_path, $is_https);
        return;
    }

    if (!function_exists('curl_init')) {
        http_response_code(500);
        header('Content-Type: application/json');
        echo json_encode(['success' => false, 'message' => 'cURL extension is required']);
        return;
    }

    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
    $response_headers = [];
    $ch = curl_init($proxy_url);

    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_TIMEOUT, 30);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, false);
    curl_setopt($ch, CURLOPT_HEADERFUNCTION, static function ($curl, string $headerLine) use (&$response_headers) {
        $trimmed = trim($headerLine);
        if ($trimmed !== '') {
            $response_headers[] = $trimmed;
        }
        return strlen($headerLine);
    });

    $forward_headers = collect_forward_headers(empty($_FILES) ? ($_SERVER['CONTENT_TYPE'] ?? null) : null);
    if (!empty($forward_headers)) {
        curl_setopt($ch, CURLOPT_HTTPHEADER, $forward_headers);
    }

    if (!empty($_SERVER['HTTP_COOKIE'])) {
        curl_setopt($ch, CURLOPT_COOKIE, $_SERVER['HTTP_COOKIE']);
    }

    if (!empty($_FILES)) {
        $post_fields = build_multipart_fields($_POST, $_FILES);
        curl_setopt($ch, CURLOPT_POSTFIELDS, $post_fields);
    } else {
        $content = file_get_contents('php://input');
        if ($content !== false && strlen($content) > 0) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, $content);
        }
    }

    $response = curl_exec($ch);
    $curl_error = curl_error($ch);
    $http_code = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
    curl_close($ch);

    if ($response === false) {
        http_response_code(502);
        header('Content-Type: application/json');
        echo json_encode(['success' => false, 'message' => 'Backend unavailable', 'details' => $curl_error]);
        return;
    }

    apply_backend_headers($response_headers, $cookie_path, $is_https);
    if ($http_code > 0) {
        http_response_code($http_code);
    }
    echo $response;
}

function proxy_api_request_via_stream(string $proxy_url, string $cookie_path, bool $is_https): void
{
    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
    $body = '';
    $content_type = $_SERVER['CONTENT_TYPE'] ?? null;

    if (!empty($_FILES)) {
        $multipart_payload = build_stream_multipart_payload($_POST, $_FILES);
        $body = $multipart_payload['body'];
        $content_type = $multipart_payload['content_type'];
    } else {
        $raw_body = file_get_contents('php://input');
        if ($raw_body !== false) {
            $body = $raw_body;
        }
    }

    $headers = collect_stream_forward_headers($content_type, strlen($body));

    $options = [
        'http' => [
            'method' => $method,
            'header' => empty($headers) ? '' : implode("\r\n", $headers) . "\r\n",
            'timeout' => 30,
            'ignore_errors' => true,
        ],
    ];

    if ($body !== '') {
        $options['http']['content'] = $body;
    }

    $context = stream_context_create($options);
    $response = @file_get_contents($proxy_url, false, $context);
    $response_headers = $http_response_header ?? [];
    $http_code = extract_http_status_code($response_headers);

    if ($response === false && $http_code === 0) {
        http_response_code(502);
        header('Content-Type: application/json');
        echo json_encode(['success' => false, 'message' => 'Backend unavailable']);
        return;
    }

    apply_backend_headers($response_headers, $cookie_path, $is_https);
    if ($http_code > 0) {
        http_response_code($http_code);
    }
    echo $response === false ? '' : $response;
}

function collect_forward_headers(?string $content_type = null): array
{
    $headers = [];

    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'HTTP_') !== 0) {
            continue;
        }

        $header = str_replace('_', ' ', substr($key, 5));
        $header = str_replace(' ', '-', ucwords(strtolower($header)));
        if (in_array($header, ['Host', 'Content-Length', 'Content-Type', 'Connection', 'Accept-Encoding', 'Transfer-Encoding'], true)) {
            continue;
        }
        $headers[] = $header . ': ' . $value;
    }

    if (!empty($content_type)) {
        $headers[] = 'Content-Type: ' . $content_type;
    }

    return $headers;
}

function collect_stream_forward_headers(?string $content_type, int $content_length): array
{
    $headers = collect_forward_headers($content_type);

    if ($content_length > 0) {
        $headers[] = 'Content-Length: ' . $content_length;
    }

    return $headers;
}

function build_multipart_fields(array $post_fields, array $files): array
{
    foreach ($files as $field_name => $file) {
        if (!isset($file['error']) || $file['error'] !== UPLOAD_ERR_OK) {
            continue;
        }

        $mime = $file['type'] ?: 'application/octet-stream';
        $name = $file['name'] ?: 'upload.bin';
        $post_fields[$field_name] = curl_file_create($file['tmp_name'], $mime, $name);
    }

    return $post_fields;
}

function build_stream_multipart_payload(array $post_fields, array $files): array
{
    $boundary = '----KuzovokProxy' . bin2hex(random_bytes(12));
    $eol = "\r\n";
    $body = '';

    foreach (flatten_form_fields($post_fields) as $field_name => $value) {
        $body .= '--' . $boundary . $eol;
        $body .= 'Content-Disposition: form-data; name="' . escape_multipart_value($field_name) . '"' . $eol . $eol;
        $body .= (string) $value . $eol;
    }

    foreach (normalize_uploaded_files($files) as $file) {
        if (($file['error'] ?? UPLOAD_ERR_NO_FILE) !== UPLOAD_ERR_OK) {
            continue;
        }
        if (empty($file['tmp_name']) || !is_readable($file['tmp_name'])) {
            continue;
        }

        $content = file_get_contents($file['tmp_name']);
        if ($content === false) {
            continue;
        }

        $mime = !empty($file['type']) ? $file['type'] : 'application/octet-stream';
        $name = !empty($file['name']) ? $file['name'] : 'upload.bin';

        $body .= '--' . $boundary . $eol;
        $body .= 'Content-Disposition: form-data; name="' . escape_multipart_value($file['field']) . '"; filename="' . escape_multipart_value($name) . '"' . $eol;
        $body .= 'Content-Type: ' . $mime . $eol . $eol;
        $body .= $content . $eol;
    }

    $body .= '--' . $boundary . '--' . $eol;

    return [
        'content_type' => 'multipart/form-data; boundary=' . $boundary,
        'body' => $body,
    ];
}

function flatten_form_fields(array $fields, string $prefix = ''): array
{
    $result = [];

    foreach ($fields as $key => $value) {
        $field_name = $prefix === '' ? (string) $key : $prefix . '[' . $key . ']';

        if (is_array($value)) {
            $result += flatten_form_fields($value, $field_name);
            continue;
        }

        $result[$field_name] = $value;
    }

    return $result;
}

function normalize_uploaded_files(array $files): array
{
    $result = [];

    foreach ($files as $field_name => $file) {
        normalize_uploaded_file_entry($result, (string) $field_name, $file);
    }

    return $result;
}

function normalize_uploaded_file_entry(array &$result, string $field_name, array $file): void
{
    if (isset($file['name']) && is_array($file['name'])) {
        foreach (array_keys($file['name']) as $key) {
            normalize_uploaded_file_entry($result, $field_name . '[' . $key . ']', [
                'name' => $file['name'][$key] ?? '',
                'type' => $file['type'][$key] ?? '',
                'tmp_name' => $file['tmp_name'][$key] ?? '',
                'error' => $file['error'][$key] ?? UPLOAD_ERR_NO_FILE,
                'size' => $file['size'][$key] ?? 0,
            ]);
        }
        return;
    }

    $file['field'] = $field_name;
    $result[] = $file;
}

function escape_multipart_value(string $value): string
{
    return str_replace(["\\", "\"", "\r", "\n"], ["\\\\", "\\\"", '', ''], $value);
}

function apply_backend_headers(array $headers, string $cookie_path, bool $is_https): void
{
    $content_type_sent = false;

    foreach ($headers as $header) {
        if (stripos($header, 'Set-Cookie:') === 0) {
            apply_backend_cookie(substr($header, strlen('Set-Cookie:')), $cookie_path, $is_https);
            continue;
        }

        if (stripos($header, 'Content-Type:') === 0) {
            if (!$content_type_sent) {
                header($header, true);
                $content_type_sent = true;
            }
            continue;
        }

        if (stripos($header, 'Transfer-Encoding:') === 0 || stripos($header, 'Content-Length:') === 0) {
            continue;
        }

        if (strpos($header, ':') !== false) {
            header($header, false);
        }
    }

    if (!$content_type_sent) {
        header('Content-Type: application/json');
    }
}

function extract_http_status_code(array $headers): int
{
    foreach ($headers as $header) {
        if (preg_match('#^HTTP/\S+\s+(\d{3})#i', $header, $matches)) {
            return (int) $matches[1];
        }
    }
    return 0;
}

function apply_backend_cookie(string $cookie_header, string $cookie_path, bool $is_https): void
{
    $parts = array_map('trim', explode(';', trim($cookie_header)));
    if (empty($parts) || strpos($parts[0], '=') === false) {
        return;
    }

    [$cookie_name, $cookie_value] = explode('=', $parts[0], 2);
    $options = [
        'path' => $cookie_path,
        'secure' => $is_https,
        'httponly' => true,
        'samesite' => 'Lax',
    ];

    foreach (array_slice($parts, 1) as $part) {
        if (strpos($part, '=') !== false) {
            [$attr_name, $attr_value] = explode('=', $part, 2);
            $attr_name = strtolower(trim($attr_name));
            $attr_value = trim($attr_value);

            if ($attr_name === 'max-age') {
                $max_age = (int) $attr_value;
                $options['expires'] = $max_age < 0 ? time() - 3600 : time() + $max_age;
            } elseif ($attr_name === 'expires') {
                $parsed = strtotime($attr_value);
                if ($parsed !== false) {
                    $options['expires'] = $parsed;
                }
            } elseif ($attr_name === 'samesite') {
                $options['samesite'] = $attr_value;
            }
        } elseif (strtolower($part) === 'httponly') {
            $options['httponly'] = true;
        }
    }

    setcookie(trim($cookie_name), $cookie_value, $options);
}

function serve_local_file(string $base_dir, string $request_uri, bool $no_store): void
{
    $base_real = realpath($base_dir);
    if ($base_real === false) {
        http_response_code(404);
        echo 'Not found';
        return;
    }

    $relative_path = ltrim($request_uri, '/');
    if (strpos($relative_path, 'img/') === 0) {
        $relative_path = substr($relative_path, 4);
    }
    $candidate = realpath($base_real . DIRECTORY_SEPARATOR . $relative_path);
    if ($candidate === false || strpos($candidate, $base_real . DIRECTORY_SEPARATOR) !== 0 || !is_file($candidate)) {
        http_response_code(404);
        echo 'Not found';
        return;
    }

    header('Content-Type: ' . (mime_content_type($candidate) ?: 'application/octet-stream'));
    if ($no_store) {
        header('Cache-Control: no-store, max-age=0');
    }
    readfile($candidate);
}
