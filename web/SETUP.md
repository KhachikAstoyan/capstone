# Frontend Setup Guide

## Stack

- **React 19** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Mantine** - Component library
- **Tailwind CSS** - Utility-first CSS framework

## Installed Packages

### Mantine
```bash
npm install @mantine/core @mantine/hooks @mantine/form @mantine/notifications @mantine/modals
```

### Tailwind CSS
```bash
npm install -D tailwindcss @tailwindcss/vite
```

## Configuration

### 1. Tailwind Config (`tailwind.config.js`)
```javascript
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

### 2. Vite Config (`vite.config.ts`)
Added Tailwind plugin:
```typescript
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
    tailwindcss(),
  ],
})
```

### 3. CSS Setup (`src/index.css`)
```css
@import url('@mantine/core/styles.css');
@import url('@mantine/notifications/styles.css');

@tailwind base;
@tailwind components;
@tailwind utilities;
```

### 4. Mantine Providers (`src/main.tsx`)
```tsx
import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { ModalsProvider } from '@mantine/modals'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <MantineProvider>
      <Notifications />
      <ModalsProvider>
        <App />
      </ModalsProvider>
    </MantineProvider>
  </StrictMode>,
)
```

## Usage

### Start Development Server
```bash
npm run dev
```

The app will be available at: http://localhost:5173/

### Build for Production
```bash
npm run build
```

### Preview Production Build
```bash
npm run preview
```

## Using Mantine + Tailwind Together

You can use both Mantine components and Tailwind utilities in the same component:

```tsx
import { Button, Card } from '@mantine/core'

function MyComponent() {
  return (
    <Card className="bg-gradient-to-r from-blue-50 to-indigo-50">
      <Button className="hover:scale-105 transition-transform">
        Click Me
      </Button>
    </Card>
  )
}
```

### Mantine Components
- Use for complex UI components (buttons, inputs, modals, etc.)
- Consistent design system
- Built-in accessibility

### Tailwind CSS
- Use for layout, spacing, and custom styling
- Utility classes for quick styling
- Responsive design utilities

## Features Configured

✅ Mantine UI Components  
✅ Tailwind CSS Utilities  
✅ Notifications System  
✅ Modals Provider  
✅ TypeScript Support  
✅ Hot Module Replacement (HMR)  
✅ React Compiler (Babel plugin)  

## Next Steps

1. Set up routing (React Router)
2. Configure API client (axios/fetch)
3. Add authentication context
4. Create layout components
5. Build feature modules

## Resources

- [Mantine Documentation](https://mantine.dev/)
- [Tailwind CSS Documentation](https://tailwindcss.com/)
- [Vite Documentation](https://vitejs.dev/)
- [React Documentation](https://react.dev/)
