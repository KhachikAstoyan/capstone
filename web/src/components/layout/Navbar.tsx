import { useState } from "react";
import { Link } from "react-router-dom";
import { Group, Button, TextInput, Avatar, Menu, Text } from "@mantine/core";
import { IconSearch, IconLogout, IconUser, IconSettings } from "@tabler/icons-react";
import { useAuth } from "../../contexts/AuthContext";
import { RegisterModal } from "../auth/RegisterModal";
import { LoginModal } from "../auth/LoginModal";

export function Navbar() {
  const { user, logout, isAuthenticated, hasPermission } = useAuth();
  const [registerOpened, setRegisterOpened] = useState(false);
  const [loginOpened, setLoginOpened] = useState(false);

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error("Logout failed:", error);
    }
  };

  return (
    <>
      <nav className="border-b border-gray-200 bg-white">
        <div className="max-w-7xl mx-auto px-6">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <Link to="/" className="no-underline">
                <Text size="xl" fw={700} className="text-blue-600 cursor-pointer hover:text-blue-700 transition-colors">
                  Capstone
                </Text>
              </Link>
            </div>

            <Group gap="md">
              <TextInput
                placeholder="Search..."
                leftSection={<IconSearch size={16} />}
                className="w-64"
              />

              {!isAuthenticated ? (
                <Group gap="sm">
                  <Button variant="subtle" onClick={() => setLoginOpened(true)}>
                    Login
                  </Button>
                  <Button onClick={() => setRegisterOpened(true)}>
                    Register
                  </Button>
                </Group>
              ) : (
                <Menu shadow="md" width={200}>
                  <Menu.Target>
                    <Avatar
                      src={user?.avatar_url}
                      alt={user?.display_name || user?.handle}
                      radius="xl"
                      className="cursor-pointer hover:ring-2 hover:ring-blue-500 transition-all"
                    >
                      {user?.display_name?.[0] || user?.handle[0]}
                    </Avatar>
                  </Menu.Target>

                  <Menu.Dropdown>
                    <Menu.Label>
                      <div>
                        <Text size="sm" fw={500}>
                          {user?.display_name || user?.handle}
                        </Text>
                        <Text size="xs" c="dimmed">
                          @{user?.handle}
                        </Text>
                      </div>
                    </Menu.Label>
                    <Menu.Divider />
                    <Menu.Item leftSection={<IconUser size={16} />}>
                      Profile
                    </Menu.Item>
                    {hasPermission('admin.access') && (
                      <Menu.Item
                        leftSection={<IconSettings size={16} />}
                        component={Link}
                        to="/admin/problems"
                      >
                        Admin Dashboard
                      </Menu.Item>
                    )}
                    <Menu.Divider />
                    <Menu.Item
                      color="red"
                      leftSection={<IconLogout size={16} />}
                      onClick={handleLogout}
                    >
                      Logout
                    </Menu.Item>
                  </Menu.Dropdown>
                </Menu>
              )}
            </Group>
          </div>
        </div>
      </nav>

      <RegisterModal
        opened={registerOpened}
        onClose={() => setRegisterOpened(false)}
      />
      <LoginModal opened={loginOpened} onClose={() => setLoginOpened(false)} />
    </>
  );
}
