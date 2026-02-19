<?php
/**
 * status.php - Get tunnel requests
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
    
    // Get tunnel requests for this node
    $tunnelStmt = $db->prepare("
        SELECT 
            t.id,
            t.source_node_id,
            t.target_node_id,
            t.local_port,
            t.remote_port,
            t.status,
            t.created_at,
            s.hostname as source_hostname,
            s.custom_name as source_name,
            d.hostname as target_hostname,
            d.custom_name as target_name
        FROM tunnels t
        LEFT JOIN nodes s ON t.source_node_id = s.id
        LEFT JOIN nodes d ON t.target_node_id = d.id
        WHERE (t.source_node_id = ? OR t.target_node_id = ?) 
        AND t.status IN ('pending', 'active')
        AND (t.expires_at IS NULL OR t.expires_at > NOW())
        ORDER BY t.created_at DESC
    ");
    $tunnelStmt->execute([$node['id'], $node['id']]);
    $tunnels = $tunnelStmt->fetchAll();
    
    sendJSON([
        'status' => 'ok',
        'tunnels' => $tunnels
    ]);
    
} catch (PDOException $e) {
    error_log("Status error: " . $e->getMessage());
    send404();
}

?>

