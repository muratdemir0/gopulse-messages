-- Sample data for testing the GoPulse Messages system
-- Run this after creating the messages table

INSERT INTO messages (recipient, content, status) VALUES
('+1234567890', 'Hello, this is test message 1', 'pending'),
('+1234567891', 'This is a longer test message to check character limit handling', 'pending'),
('+1234567892', 'Short message', 'pending'),
('+1987654321', 'Welcome to GoPulse Messages!', 'pending'),
('+1555666777', 'Testing automatic message sending system', 'pending'),
('+1444555666', 'Message with special chars: àáâã', 'pending'),
('+1333444555', 'Test message for retry logic', 'pending'),
('+1222333444', 'Sample notification message', 'pending');

-- You can also insert some already sent messages for testing the list endpoints
INSERT INTO messages (recipient, content, status, sent_at, response_id, response_code) VALUES
('+1111222333', 'This message was already sent', 'sent', NOW() - INTERVAL '1 hour', 'webhook-msg-001', 200),
('+1000111222', 'Another sent message', 'sent', NOW() - INTERVAL '2 hours', 'webhook-msg-002', 200);

-- Insert a failed message for testing
INSERT INTO messages (recipient, content, status, retry_count, error_message, last_attempt_at) VALUES
('+1999888777', 'This message failed to send', 'failed', 3, 'Webhook endpoint returned 500', NOW() - INTERVAL '30 minutes');