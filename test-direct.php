<?php
// Прямой тест прокси
error_log("Direct test started");

$backend_url = 'http://127.0.0.1:8080';
$test_data = '{"username":"test","password":"test"}';

$ch = curl_init($backend_url . '/api/login');
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $test_data);
curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
curl_setopt($ch, CURLOPT_TIMEOUT, 10);

$response = curl_exec($ch);
$error = curl_error($ch);
$http_code = curl_getinfo($ch, CURLINFO_HTTP_CODE);

echo "HTTP Code: $http_code\n";
echo "Error: " . ($error ?: 'none') . "\n";
echo "Response: $response\n";

curl_close($ch);
?>
