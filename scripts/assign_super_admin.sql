-- Script to assign super_admin role to a user
-- Replace 'your_email@example.com' with your actual email

-- First, let's see your user ID
SELECT id, handle, email FROM users WHERE email = 'your_email@example.com';

-- Get the super_admin role ID
SELECT id, name FROM roles WHERE name = 'super_admin';

-- Assign super_admin role to your user
-- Replace the UUIDs below with the actual IDs from the queries above
INSERT INTO user_roles (user_id, role_id, granted_at)
SELECT 
    u.id as user_id,
    r.id as role_id,
    NOW() as granted_at
FROM users u, roles r
WHERE u.email = 'your_email@example.com'
  AND r.name = 'super_admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Verify the assignment
SELECT 
    u.handle,
    u.email,
    r.name as role_name
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE u.email = 'your_email@example.com';
