<?php
/**
 * Ametie Server Configuration
 */

// Database configuration
define('DB_HOST', 'localhost');
define('DB_NAME', 'ametie');
define('DB_USER', 'ametie_user');
define('DB_PASS', 'change_me_password');
define('DB_CHARSET', 'utf8mb4');

// Security settings
define('API_KEY_FAILURE_THRESHOLD', 9); // Number of failed attempts before revealing auth failure
define('TIMESTAMP_WINDOW', 300); // 5 minutes in seconds
define('DEFAULT_TTL_DAYS', 7); // Default TTL for auto-cleanup

// Get database connection
function getDB() {
    static $pdo = null;
    if ($pdo === null) {
        try {
            $dsn = "mysql:host=" . DB_HOST . ";dbname=" . DB_NAME . ";charset=" . DB_CHARSET;
            $options = [
                PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
                PDO::ATTR_EMULATE_PREPARES => false,
            ];
            $pdo = new PDO($dsn, DB_USER, DB_PASS, $options);
        } catch (PDOException $e) {
            error_log("Database connection failed: " . $e->getMessage());
            return null;
        }
    }
    return $pdo;
}

// Get client IP address
function getClientIP() {
    $ipkeys = ['HTTP_CLIENT_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'];
    foreach ($ipkeys as $keyword) {
        if (array_key_exists($keyword, $_SERVER) && !empty($_SERVER[$keyword])) {
            $ip = $_SERVER[$keyword];
            if (strpos($ip, ',') !== false) {
                $ip = explode(',', $ip)[0];
            }
            $ip = trim($ip);
            if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_NO_PRIV_RANGE | FILTER_FLAG_NO_RES_RANGE)) {
                return $ip;
            }
        }
    }
    return $_SERVER['REMOTE_ADDR'] ?? '0.0.0.0';
}

// Track failed authentication attempt
function trackFailedAttempt($ip, $apiKeyHash = null) {
    $db = getDB();
    if (!$db) return;
    
    try {
        $stmt = $db->prepare("
            INSERT INTO failed_attempts (ip_address, api_key_hash, attempt_count, last_attempt, first_attempt)
            VALUES (?, ?, 1, NOW(), NOW())
            ON DUPLICATE KEY UPDATE
                attempt_count = attempt_count + 1,
                last_attempt = NOW()
        ");
        $stmt->execute([$ip, $apiKeyHash]);
    } catch (PDOException $e) {
        error_log("Failed to track attempt: " . $e->getMessage());
    }
}

// Check if IP has exceeded failure threshold
function hasExceededThreshold($ip) {
    $db = getDB();
    if (!$db) return false;
    
    try {
        $stmt = $db->prepare("SELECT attempt_count FROM failed_attempts WHERE ip_address = ?");
        $stmt->execute([$ip]);
        $result = $stmt->fetch();
        return $result && $result['attempt_count'] >= API_KEY_FAILURE_THRESHOLD;
    } catch (PDOException $e) {
        return false;
    }
}

// Get encryption key for API keys (in production, use environment variable or secure storage)
function getEncryptionKey() {
    // In production, this should come from environment variable or secure key management
    $key = getenv('AMETIE_ENCRYPTION_KEY');
    if (empty($key)) {
        // Fallback: use a default key (NOT SECURE - for development only)
        $key = 'CHANGE_ME_IN_PRODUCTION_' . hash('sha256', __FILE__);
    }
    return substr(hash('sha256', $key), 0, 32); // 32 bytes for AES-256
}

// Encrypt API key for storage
function encryptAPIKey($apiKey) {
    $key = getEncryptionKey();
    $iv = openssl_random_pseudo_bytes(16);
    $encrypted = openssl_encrypt($apiKey, 'AES-256-CBC', $key, 0, $iv);
    return base64_encode($iv . $encrypted);
}

// Decrypt API key from storage
function decryptAPIKey($encryptedKey) {
    $key = getEncryptionKey();
    $data = base64_decode($encryptedKey);
    $iv = substr($data, 0, 16);
    $encrypted = substr($data, 16);
    return openssl_decrypt($encrypted, 'AES-256-CBC', $key, 0, $iv);
}

// Verify request signature
function verifyRequest($requestBody, $timestamp, $nonce, $nodeId, $signature, $endpointPath) {
    $db = getDB();
    if (!$db) return false;
    
    // Validate timestamp (within 5 minute window)
    $now = time();
    if (abs($now - $timestamp) > TIMESTAMP_WINDOW) {
        return false;
    }
    
    // Lookup API key by node_id
    $stmt = $db->prepare("
        SELECT ak.key_hash, ak.encrypted_key, n.id as node_id
        FROM nodes n
        JOIN api_keys ak ON ak.node_id = n.id
        WHERE n.node_id_hash = ? AND ak.is_active = 1
        AND (ak.expires_at IS NULL OR ak.expires_at > NOW())
        ORDER BY ak.last_used DESC
        LIMIT 1
    ");
    $stmt->execute([$nodeId]);
    $keyData = $stmt->fetch();
    
    if (!$keyData) {
        return false;
    }
    
    // Decrypt API key
    $apiKey = null;
    if (!empty($keyData['encrypted_key'])) {
        $apiKey = decryptAPIKey($keyData['encrypted_key']);
    } else {
        // Fallback: if no encrypted key, we can't verify (should not happen in production)
        error_log("No encrypted key found for node_id: " . $nodeId);
        return false;
    }
    
    if (empty($apiKey)) {
        return false;
    }
    
    // Reconstruct expected signature
    $signatureData = $requestBody . $timestamp . $nonce . $endpointPath;
    $expectedSignature = hash_hmac('sha256', $signatureData, $apiKey, true);
    
    // Compare signatures (constant-time comparison to prevent timing attacks)
    if (strlen($signature) !== strlen($expectedSignature)) {
        return false;
    }
    
    $result = 0;
    for ($i = 0; $i < strlen($signature); $i++) {
        $result |= ord($signature[$i]) ^ ord($expectedSignature[$i]);
    }
    
    if ($result !== 0) {
        return false;
    }
    
    // Update last_used
    $updateStmt = $db->prepare("UPDATE api_keys SET last_used = NOW() WHERE key_hash = ?");
    $updateStmt->execute([$keyData['key_hash']]);
    
    return true;
}

// Decrypt request body (XOR with derived key)
function decryptRequestBody($encryptedBody, $apiKey, $timestamp, $nonce) {
    $key = hash_hmac('sha256', $apiKey, $timestamp . $nonce, true);
    $key = substr($key, 0, 32);
    
    $decrypted = '';
    $bodyLen = strlen($encryptedBody);
    for ($i = 0; $i < $bodyLen; $i++) {
        $decrypted .= chr(ord($encryptedBody[$i]) ^ ord($key[$i % 32]));
    }
    return $decrypted;
}

// Send JSON response
function sendJSON($data, $statusCode = 200) {
    http_response_code($statusCode);
    header('Content-Type: application/json');
    echo json_encode($data);
    exit;
}

// Send 404 (for obfuscation)
function send404() {
    http_response_code(404);
    header('Content-Type: text/html');
    echo '<!DOCTYPE html><html><head><title>404 Not Found</title></head><body><h1>404 Not Found</h1></body></html>';
    exit;
}

// Common request validation
function validateRequest() {
    $ip = getClientIP();
    
    // Get headers (handle rotation)
    $requestId = $_SERVER['HTTP_X_REQUEST_ID'] ?? $_SERVER['HTTP_X_REQ_ID'] ?? null;
    $clientTime = $_SERVER['HTTP_X_CLIENT_TIME'] ?? $_SERVER['HTTP_X_TIME'] ?? null;
    $nodeId = $_SERVER['HTTP_X_NODE_ID'] ?? $_SERVER['HTTP_X_NODE'] ?? null;
    
    if (!$requestId || !$clientTime || !$nodeId) {
        trackFailedAttempt($ip);
        send404();
    }
    
    $timestamp = intval($clientTime);
    $nonce = base64_decode($requestId);
    
    if (!$nonce || strlen($nonce) < 12) {
        trackFailedAttempt($ip);
        send404();
    }
    
    // Get request body
    $requestBody = file_get_contents('php://input');
    $data = json_decode($requestBody, true);
    
    if (!$data || !isset($data['sig']) || !isset($data['chk'])) {
        trackFailedAttempt($ip);
        if (!hasExceededThreshold($ip)) {
            send404();
        } else {
            sendJSON(['error' => 'Invalid request'], 400);
        }
    }
    
    $signature = base64_decode($data['sig']);
    $endpointPath = $_SERVER['REQUEST_URI'] ?? '';
    
    // Verify signature
    if (!verifyRequest($requestBody, $timestamp, $nonce, $nodeId, $signature, $endpointPath)) {
        trackFailedAttempt($ip, hash('sha256', $nodeId));
        if (!hasExceededThreshold($ip)) {
            send404();
        } else {
            sendJSON(['error' => 'Authentication failed'], 401);
        }
    }
    
    return [
        'node_id' => $nodeId,
        'timestamp' => $timestamp,
        'nonce' => $nonce,
        'body' => $data
    ];
}

?>

