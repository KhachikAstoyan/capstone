#!/bin/bash

# Script to assign super_admin role to a user
# Usage: ./assign_super_admin.sh your_email@example.com

if [ -z "$1" ]; then
    echo "Usage: $0 <user_email>"
    echo "Example: $0 admin@example.com"
    exit 1
fi

USER_EMAIL="$1"

# Load environment variables if .env exists
if [ -f .env ]; then
    source .env
fi

# Use the database URL from environment or default
DB_URL="${API_DATABASE_URL:-postgresql://capstone:capstone@localhost:5432/capstone?sslmode=disable}"

echo "Assigning super_admin role to user: $USER_EMAIL"
echo ""

psql "$DB_URL" <<EOF
-- Check if user exists
DO \$\$
DECLARE
    v_user_id UUID;
    v_role_id UUID;
BEGIN
    -- Get user ID
    SELECT id INTO v_user_id FROM users WHERE email = '$USER_EMAIL';
    
    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'User with email % not found', '$USER_EMAIL';
    END IF;
    
    -- Get super_admin role ID
    SELECT id INTO v_role_id FROM roles WHERE name = 'super_admin';
    
    IF v_role_id IS NULL THEN
        RAISE EXCEPTION 'super_admin role not found. Run migrations and seed first.';
    END IF;
    
    -- Assign role
    INSERT INTO user_roles (user_id, role_id, granted_at)
    VALUES (v_user_id, v_role_id, NOW())
    ON CONFLICT (user_id, role_id) DO NOTHING;
    
    RAISE NOTICE 'Successfully assigned super_admin role to %', '$USER_EMAIL';
END \$\$;

-- Show user's roles
SELECT 
    u.handle,
    u.email,
    r.name as role_name,
    ur.granted_at
FROM users u
JOIN user_roles ur ON u.id = ur.user_id
JOIN roles r ON ur.role_id = r.id
WHERE u.email = '$USER_EMAIL';
EOF

echo ""
echo "Done! Please log out and log back in to refresh your permissions."
