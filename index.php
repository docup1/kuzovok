<?php
/**
 * Кузовок - PHP прокси для Go бэкенда
 */

$backend_url = getenv('KUSOVOK_BACKEND_URL') ?: 'http://127.0.0.1:8080';
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
    proxy_api_request($backend_url . $request_uri, $cookie_path, $is_https);
    exit;
}

if (strpos($request_uri, '/img/') === 0) {
    serve_local_file($image_dir, $request_uri, true);
    exit;
}

if ($request_uri === '/' || $request_uri === '/index.html') {
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

function proxy_api_request(string $proxy_url, string $cookie_path, bool $is_https): void
{
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

    $forward_headers = collect_forward_headers();
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

function collect_forward_headers(): array
{
    $headers = [];

    foreach ($_SERVER as $key => $value) {
        if (strpos($key, 'HTTP_') !== 0) {
            continue;
        }

        $header = str_replace('_', ' ', substr($key, 5));
        $header = str_replace(' ', '-', ucwords(strtolower($header)));
        if (in_array($header, ['Host', 'Content-Length', 'Content-Type'], true)) {
            continue;
        }
        $headers[] = $header . ': ' . $value;
    }

    if (empty($_FILES) && !empty($_SERVER['CONTENT_TYPE'])) {
        $headers[] = 'Content-Type: ' . $_SERVER['CONTENT_TYPE'];
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
