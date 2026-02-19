-- Ametie Database Schema
-- MySQL/MariaDB compatible

CREATE DATABASE IF NOT EXISTS ametie CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE ametie;

-- Nodes table
CREATE TABLE IF NOT EXISTS nodes (
    id INT AUTO_INCREMENT PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    custom_name VARCHAR(255),
    node_id_hash VARCHAR(64) NOT NULL UNIQUE,
    last_heartbeat DATETIME,
    status ENUM('active', 'inactive', 'offline') DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_node_id_hash (node_id_hash),
    INDEX idx_hostname (hostname),
    INDEX idx_last_heartbeat (last_heartbeat)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id INT AUTO_INCREMENT PRIMARY KEY,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    encrypted_key TEXT NOT NULL,
    node_id INT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME,
    is_active BOOLEAN DEFAULT TRUE,
    expires_at DATETIME NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    INDEX idx_key_hash (key_hash),
    INDEX idx_node_id (node_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Commands table
CREATE TABLE IF NOT EXISTS commands (
    id INT AUTO_INCREMENT PRIMARY KEY,
    node_id INT NOT NULL,
    command TEXT NOT NULL,
    status ENUM('pending', 'executing', 'completed', 'failed', 'cancelled') DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    executed_at DATETIME NULL,
    expires_at DATETIME NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    INDEX idx_node_id (node_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tunnels table
CREATE TABLE IF NOT EXISTS tunnels (
    id INT AUTO_INCREMENT PRIMARY KEY,
    source_node_id INT NOT NULL,
    target_node_id INT NOT NULL,
    local_port INT,
    remote_port INT,
    status ENUM('pending', 'active', 'closed', 'failed') DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NULL,
    FOREIGN KEY (source_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (target_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    INDEX idx_source_node (source_node_id),
    INDEX idx_target_node (target_node_id),
    INDEX idx_status (status),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Mailbox table (for command output)
CREATE TABLE IF NOT EXISTS mailbox (
    id INT AUTO_INCREMENT PRIMARY KEY,
    node_id INT NOT NULL,
    command_id INT NULL,
    message_type ENUM('command_output', 'notification', 'error') DEFAULT 'command_output',
    content LONGTEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NULL,
    FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (command_id) REFERENCES commands(id) ON DELETE SET NULL,
    INDEX idx_node_id (node_id),
    INDEX idx_command_id (command_id),
    INDEX idx_created_at (created_at),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Messages table (for node-to-node messaging)
CREATE TABLE IF NOT EXISTS messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    from_node_id INT NOT NULL,
    to_node_id INT NOT NULL,
    message TEXT NOT NULL,
    is_read BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME NULL,
    expires_at DATETIME NULL,
    FOREIGN KEY (from_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    FOREIGN KEY (to_node_id) REFERENCES nodes(id) ON DELETE CASCADE,
    INDEX idx_to_node (to_node_id),
    INDEX idx_from_node (from_node_id),
    INDEX idx_is_read (is_read),
    INDEX idx_created_at (created_at),
    INDEX idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Failed attempts tracking
CREATE TABLE IF NOT EXISTS failed_attempts (
    id INT AUTO_INCREMENT PRIMARY KEY,
    ip_address VARCHAR(45) NOT NULL,
    api_key_hash VARCHAR(64),
    attempt_count INT DEFAULT 1,
    last_attempt DATETIME DEFAULT CURRENT_TIMESTAMP,
    first_attempt DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ip_address (ip_address),
    INDEX idx_api_key_hash (api_key_hash),
    INDEX idx_last_attempt (last_attempt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Cleanup procedure (to be run periodically)
DELIMITER //
CREATE PROCEDURE IF NOT EXISTS cleanup_expired_data()
BEGIN
    DELETE FROM commands WHERE expires_at IS NOT NULL AND expires_at < NOW();
    DELETE FROM tunnels WHERE expires_at IS NOT NULL AND expires_at < NOW();
    DELETE FROM mailbox WHERE expires_at IS NOT NULL AND expires_at < NOW();
    DELETE FROM messages WHERE expires_at IS NOT NULL AND expires_at < NOW();
    DELETE FROM failed_attempts WHERE last_attempt < DATE_SUB(NOW(), INTERVAL 24 HOUR);
END //
DELIMITER ;

