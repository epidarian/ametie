<?php
/**
 * list.php - List nodes endpoint (for CLI)
 * Obscured name to avoid detection
 */

require_once __DIR__ . '/config.php';

header('Content-Type: application/json');

if ($_SERVER['REQUEST_METHOD'] !== 'GET') {
    send404();
}

$validated = validateRequest();
$db = getDB();

if (!$db) {
    sendJSON(['error' => 'Service unavailable'], 503);
}

try {
    $stmt = $db->prepare("
        SELECT 
            n.id,
            n.hostname,
            n.custom_name,
            n.last_heartbeat,
            n.status,
            n.created_at
        FROM nodes n
        WHERE n.status = 'active'
        ORDER BY n.last_heartbeat DESC
    ");
    $stmt->execute();
    $nodes = $stmt->fetchAll();
    
    sendJSON([
        'status' => 'ok',
        'nodes' => $nodes
    ]);
    
} catch (PDOException $e) {
    error_log("List error: " . $e->getMessage());
    send404();
}

?>

