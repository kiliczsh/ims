-- Sample data for testing Insider Message Sender
-- Run this after setting up the database

INSERT INTO messages (phone_number, content, status) VALUES
    ('+905551234567', 'Hello from Insider! This is a test message.', 'pending'),
    ('+905559876543', 'Insider - Project test message', 'pending'),
    ('+905551111111', 'Welcome to our messaging service!', 'pending'),
    ('+905552222222', 'Another test message for the queue', 'pending'),
    ('+905553333333', 'SMS testing with Insider platform', 'pending');

-- Check the inserted data
SELECT id, phone_number, content, status, created_at FROM messages ORDER BY created_at; 