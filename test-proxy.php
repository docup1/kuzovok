<?php
// Тест прокси
$backend_url = 'http://127.0.0.1:8080';
$test_url = $backend_url . '/api/me';

echo "Testing: $test_url\n\n";

$ch = curl_init($test_url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_TIMEOUT, 10);

$response = curl_exec($ch);
$error = curl_error($ch);
$http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);

echo "HTTP Code: $http_code\n";
echo "Error: $error\n";
echo "Response: $response\n";

curl_close($ch);
?>
