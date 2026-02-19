<?php
/**
 * request.php - Register tunnel request
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
    
    // Get source node
    $stmt = $db->prepare("SELECT id FROM nodes WHERE node_id_hash = ?");
    $stmt->execute([$nodeId]);
    $sourceNode = $stmt->fetch();
    
    if (!$sourceNode) {
        send404();
    }
    
    // Get target node
    $targetNode = $body['target_node'] ?? null;
    $localPort = $body['local_port'] ?? null;
    $remotePort = $body['remote_port'] ?? null;
    $ttlDays = $body['ttl_days'] ?? DEFAULT_TTL_DAYS;
    
    if (!$targetNode || !$localPort || !$remotePort) {
        sendJSON(['error' => 'Missing required fields'], 400);
    }
    
    // Find target node
    $targetStmt = $db->prepare("
        SELECT id FROM nodes 
        WHERE (hostname = ? OR custom_name = ?) AND status = 'active'
        LIMIT 1
    ");
    $targetStmt->execute([$targetNode, $targetNode]);
    $target = $targetStmt->fetch();
    
    if (!$target) {
        sendJSON(['error' => 'Target node not found'], 404);
    }
    
    // Insert tunnel request
    $insertStmt = $db->prepare("
        INSERT INTO tunnels (source_node_id, target_node_id, local_port, remote_port, status, expires_at)
        VALUES (?, ?, ?, ?, 'pending', DATE_ADD(NOW(), INTERVAL ? DAY))
    ");
    $insertStmt->execute([$sourceNode['id'], $target['id'], $localPort, $remotePort, $ttlDays]);
    $tunnelId = $db->lastInsertId();
    
    sendJSON([
        'status' => 'ok',
        'tunnel_id' => $tunnelId,
        'message' => 'Tunnel request registered'
    ]);
    
} catch (PDOException $e) {
    error_log("Request error: " . $e->getMessage());
    send404();
}

?>

