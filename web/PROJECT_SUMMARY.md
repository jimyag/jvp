# JVP Admin Dashboard - Project Summary

## Overview

A modern, production-ready admin dashboard for JVP (jimyag's Virtualization Platform) built with Next.js 15, React 19, TypeScript, and Tailwind CSS. The design is inspired by MotherDuck's clean and professional aesthetic.

## Technology Stack

- **Framework**: Next.js 15.5.6 (App Router)
- **UI Library**: React 19
- **Styling**: Tailwind CSS 3.4.1
- **Language**: TypeScript 5
- **Icons**: Lucide React 0.456.0
- **Build Tool**: Next.js built-in compiler

## Design System

### Colors
```css
Primary: #383838 (dark gray)
Accent: #6FC2FF (bright blue)
Accent Dark: #2BA5FF
Background: #F4EFEA (warm beige)
Card Background: #FFFFFF
Yellow: #FFDE00
Coral: #FF7169
```

### Typography
- **Font Family**: Inter (sans-serif)
- **Monospace**: System monospace stack

### Design Principles
1. **Clean & Minimal**: Ample whitespace, clear visual hierarchy
2. **Border-based**: Components use borders instead of heavy shadows
3. **Smooth Transitions**: 200ms transitions on interactive elements
4. **Responsive**: Mobile-first approach, works on all screen sizes
5. **Accessible**: Semantic HTML, clear labels, keyboard navigation

## File Structure

```
web/
├── app/                          # Next.js App Router
│   ├── instances/page.tsx       # VM instances management
│   ├── volumes/page.tsx         # Storage volumes management
│   ├── images/page.tsx          # System images management
│   ├── keypairs/page.tsx        # SSH key pairs management
│   ├── layout.tsx               # Root layout with Inter font
│   ├── page.tsx                 # Home (redirects to /instances)
│   └── globals.css              # Global styles & utilities
│
├── components/                   # Reusable React components
│   ├── DashboardLayout.tsx      # Main layout with sidebar
│   ├── Sidebar.tsx              # Navigation sidebar
│   ├── Header.tsx               # Page header component
│   ├── Table.tsx                # Data table component
│   ├── Modal.tsx                # Modal dialog component
│   └── StatusBadge.tsx          # Status indicator badge
│
├── public/                       # Static assets
│   └── favicon.ico
│
├── next.config.ts               # Next.js config with API proxy
├── tailwind.config.ts           # Tailwind custom theme
├── tsconfig.json                # TypeScript configuration
├── postcss.config.mjs           # PostCSS with Tailwind
├── package.json                 # Dependencies
├── .eslintrc.json              # ESLint configuration
├── .gitignore                  # Git ignore rules
├── README.md                    # Documentation
├── QUICKSTART.md               # Quick start guide
└── PROJECT_SUMMARY.md          # This file
```

## Features

### 1. Instances Management
- **List View**: Table showing all VM instances with status
- **Create**: Modal form with CPU, memory, disk, image, and keypair options
- **Actions**: Start, stop, restart, delete
- **Real-time Status**: Color-coded status badges

### 2. Volumes Management
- **List View**: Table showing volumes with size and attachment status
- **Create**: Modal form for new volumes
- **Attach/Detach**: Attach volumes to instances, detach when needed
- **Delete**: Remove unused volumes

### 3. Images Management
- **List View**: Table showing available system images
- **Register**: Modal form to register images from URLs
- **Details**: OS type, size, creation date
- **Delete**: Remove images from the system

### 4. Key Pairs Management
- **List View**: Table showing SSH key pairs with fingerprints
- **Create**: Generate new RSA or Ed25519 key pairs
- **Import**: Import existing public keys
- **Download**: Download private keys (shown only once)
- **Delete**: Remove key pairs

## API Integration

The dashboard proxies all `/api/*` requests to the JVP backend at `http://localhost:8080`.

### Expected API Endpoints

```
GET    /api/instances          # List instances
POST   /api/instances          # Create instance
POST   /api/instances/:id/start
POST   /api/instances/:id/stop
POST   /api/instances/:id/restart
DELETE /api/instances/:id

GET    /api/volumes            # List volumes
POST   /api/volumes            # Create volume
POST   /api/volumes/:id/attach
POST   /api/volumes/:id/detach
DELETE /api/volumes/:id

GET    /api/images             # List images
POST   /api/images/register    # Register image
DELETE /api/images/:id

GET    /api/keypairs           # List key pairs
POST   /api/keypairs           # Create key pair
POST   /api/keypairs/import    # Import key pair
DELETE /api/keypairs/:name
```

## Component Details

### Table Component
Generic, reusable table with:
- Custom column definitions
- Render functions for custom cell content
- Empty state support
- Hover effects
- Responsive layout

### Modal Component
Flexible modal dialog with:
- Backdrop click to close
- Close button
- Custom width sizes (sm, md, lg, xl)
- Body scroll lock
- Smooth animations

### Sidebar Component
Navigation sidebar with:
- Mobile hamburger menu
- Active route highlighting
- Smooth transitions
- Fixed positioning
- Responsive overlay

## Styling Utilities

Custom Tailwind utilities in `globals.css`:

```css
.btn              # Base button
.btn-primary      # Primary action
.btn-secondary    # Secondary action
.btn-danger       # Destructive action
.card             # Card container
.input            # Form input
.label            # Form label
```

## Build Output

```
Route (app)                     Size    First Load JS
┌ ○ /                          329 B   102 kB
├ ○ /images                    3.38 kB 110 kB
├ ○ /instances                 3.47 kB 110 kB
├ ○ /keypairs                  3.6 kB  110 kB
└ ○ /volumes                   3.58 kB 110 kB
```

All routes are pre-rendered as static content for optimal performance.

## Performance Optimizations

1. **Static Generation**: All pages pre-rendered at build time
2. **Code Splitting**: Automatic route-based splitting
3. **Optimized Images**: Next.js Image component ready
4. **Font Optimization**: Inter font automatically optimized
5. **CSS Purging**: Tailwind removes unused CSS in production

## Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)
- Mobile browsers (iOS Safari, Chrome Mobile)

## Development Workflow

```bash
# Install dependencies
npm install

# Start dev server (hot reload enabled)
npm run dev

# Type checking
npx tsc --noEmit

# Linting
npm run lint

# Build for production
npm run build

# Start production server
npm start
```

## Future Enhancements

Potential features to add:

1. **Authentication**: Login/logout, user sessions
2. **Networking**: Network and security group management
3. **Monitoring**: Real-time metrics and charts
4. **Logs**: View instance console logs
5. **Notifications**: Toast notifications for actions
6. **Search & Filter**: Advanced filtering in tables
7. **Bulk Actions**: Select multiple items for batch operations
8. **Dark Mode**: Toggle between light and dark themes
9. **Websockets**: Real-time status updates

## Customization

### Changing Colors
Edit `tailwind.config.ts`:
```typescript
colors: {
  primary: { DEFAULT: '#YOUR_COLOR' },
  accent: { DEFAULT: '#YOUR_COLOR' },
  // ...
}
```

### Adding New Pages
1. Create `app/your-page/page.tsx`
2. Add route to `components/Sidebar.tsx`
3. Follow existing page patterns

### Modifying API Endpoint
Edit `next.config.ts`:
```typescript
destination: 'http://your-api-url/api/:path*'
```

## Credits

- **Design Inspiration**: MotherDuck (https://motherduck.com)
- **Icons**: Lucide React
- **Fonts**: Inter (Google Fonts)
- **Framework**: Next.js by Vercel

## License

Part of the JVP project by jimyag.
