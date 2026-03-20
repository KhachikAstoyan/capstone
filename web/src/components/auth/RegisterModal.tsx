import { Modal, TextInput, PasswordInput, Button, Stack } from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import type { RegisterRequest } from '../../types/auth';

interface RegisterModalProps {
  opened: boolean;
  onClose: () => void;
}

export function RegisterModal({ opened, onClose }: RegisterModalProps) {
  const { register } = useAuth();

  const form = useForm<RegisterRequest>({
    initialValues: {
      handle: '',
      email: '',
      password: '',
      display_name: '',
    },
    validate: {
      handle: (value) => (!value ? 'Handle is required' : null),
      email: (value) => (!value ? 'Email is required' : !/^\S+@\S+$/.test(value) ? 'Invalid email' : null),
      password: (value) => (!value ? 'Password is required' : value.length < 8 ? 'Password must be at least 8 characters' : null),
    },
  });

  const handleSubmit = async (values: RegisterRequest) => {
    try {
      const submitData = {
        ...values,
        display_name: values.display_name || undefined,
      };
      await register(submitData);
      notifications.show({
        title: 'Success',
        message: 'Account created successfully!',
        color: 'teal',
      });
      onClose();
      form.reset();
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: error instanceof Error ? error.message : 'Failed to register',
        color: 'red',
      });
    }
  };

  return (
    <Modal opened={opened} onClose={onClose} title="Create Account" centered>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="md">
          <TextInput
            label="Handle"
            placeholder="username"
            required
            {...form.getInputProps('handle')}
          />
          <TextInput
            label="Email"
            placeholder="your@email.com"
            type="email"
            required
            {...form.getInputProps('email')}
          />
          <TextInput
            label="Display Name"
            placeholder="Your Name (optional)"
            {...form.getInputProps('display_name')}
          />
          <PasswordInput
            label="Password"
            placeholder="••••••••"
            required
            {...form.getInputProps('password')}
          />
          <Button type="submit" fullWidth>
            Register
          </Button>
        </Stack>
      </form>
    </Modal>
  );
}
