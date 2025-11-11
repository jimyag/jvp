# JVP Admin Dashboard

A modern, responsive admin dashboard for JVP (jimyag's Virtualization Platform) built with Next.js and Tailwind CSS, inspired by MotherDuck's design system.

## Features

- **Instances Management**: Create, start, stop, restart, and delete virtual machine instances
- **Volumes Management**: Create, attach, detach, and delete block storage volumes
- **Images Management**: Register and manage system images
- **Key Pairs Management**: Create, import, and manage SSH key pairs

## Design System

The dashboard follows MotherDuck's design principles:

- **Clean, minimal aesthetic** with ample whitespace
- **Border-based design language** with subtle hover effects
- **Professional color palette**:
  - Primary: `#383838` (dark gray)
  - Accent: `#6FC2FF` (bright blue)
  - Background: `#F4EFEA` (warm beige)
- **Typography**: Inter for body text, monospace for code
- **Responsive layout** with mobile-first approach

## Getting Started

### Prerequisites

- Node.js 18+ and npm/yarn
- JVP backend API running on `http://192.168.2.100:8080`

### Installation

```bash
# Install dependencies
npm install

# Run development server
npm run dev

# Build for production
npm run build

# Start production server
npm start
```

The dashboard will be available at `http://localhost:3000`.

### API Configuration

The dashboard is configured to proxy API requests to the JVP backend. Currently configured to `http://192.168.2.100:8080`.

To change the backend URL, edit `next.config.ts`:

```typescript
async rewrites() {
  return [
    {
      source: '/api/:path*',
      destination: 'http://192.168.2.100:8080/api/:path*',
    },
  ];
}
```

### API Endpoints

All API endpoints use POST method with JSON request body:

**Instances:**
- `POST /api/instances/describe` - List all instances
- `POST /api/instances/run` - Create new instance
- `POST /api/instances/start` - Start instance(s)
- `POST /api/instances/stop` - Stop instance(s)
- `POST /api/instances/restart` - Restart instance(s)
- `POST /api/instances/terminate` - Delete instance(s)
- `POST /api/instances/reset-password` - Reset instance password

**Volumes:**
- `POST /api/volumes/describe` - List all volumes
- `POST /api/volumes/create` - Create new volume
- `POST /api/volumes/attach` - Attach volume to instance
- `POST /api/volumes/detach` - Detach volume from instance
- `POST /api/volumes/delete` - Delete volume

**Images:**
- `POST /api/images/describe` - List all images
- `POST /api/images/register` - Register new image
- `POST /api/images/deregister` - Deregister image

**Key Pairs:**
- `POST /api/keypairs/describe` - List all key pairs
- `POST /api/keypairs/create` - Create new key pair
- `POST /api/keypairs/import` - Import existing public key
- `POST /api/keypairs/delete` - Delete key pair

## Project Structure

```
web/
├── app/                    # Next.js app directory
│   ├── instances/         # Instances management page
│   ├── volumes/           # Volumes management page
│   ├── images/            # Images management page
│   ├── keypairs/          # Key pairs management page
│   ├── layout.tsx         # Root layout
│   ├── page.tsx           # Home page (redirects to instances)
│   └── globals.css        # Global styles
├── components/            # Reusable components
│   ├── DashboardLayout.tsx
│   ├── Sidebar.tsx
│   ├── Header.tsx
│   ├── Table.tsx
│   ├── Modal.tsx
│   └── StatusBadge.tsx
├── public/               # Static assets
├── next.config.ts        # Next.js configuration
├── tailwind.config.ts    # Tailwind CSS configuration
└── package.json          # Dependencies
```

## Components

### DashboardLayout
Main layout wrapper with sidebar navigation.

### Table
Reusable table component with customizable columns and renderers.

### Modal
Responsive modal dialog for forms and confirmations.

### StatusBadge
Colored badge for displaying status (running, stopped, pending, etc.).

### Header
Page header with title, description, and action buttons.

## Styling

The project uses Tailwind CSS with custom utility classes:

- `.btn` - Base button style
- `.btn-primary` - Primary action button
- `.btn-secondary` - Secondary action button
- `.btn-danger` - Destructive action button
- `.card` - Card container with hover effect
- `.input` - Form input field
- `.label` - Form label

## Development

### Adding a New Page

1. Create a new directory in `app/`
2. Add a `page.tsx` file
3. Import `DashboardLayout` and wrap your content
4. Add navigation link in `components/Sidebar.tsx`

Example:

```typescript
"use client";

import DashboardLayout from "@/components/DashboardLayout";
import Header from "@/components/Header";

export default function NewPage() {
  return (
    <DashboardLayout>
      <Header title="New Page" description="Description" />
      {/* Your content */}
    </DashboardLayout>
  );
}
```

## Technologies

- **Next.js 15** - React framework
- **React 19** - UI library
- **Tailwind CSS 3** - Utility-first CSS framework
- **TypeScript** - Type safety
- **Lucide React** - Icon library

## Data Models and Units

### Field Names

The frontend uses exact field names from backend entities:

**Instance:**
- `id`, `name`, `state` (not `status`)
- `vcpus` (number of CPU cores)
- `memory_mb` (memory in MB)
- `image_id`, `volume_id`

**Volume:**
- `volumeID` (not `id`)
- `sizeGB` (size in GB)
- `state` (not `status`)
- `volumeType`, `attachments`

**Image:**
- `id`, `name`, `state`
- `size_gb` (size in GB)
- `format` (qcow2, raw, etc.)

**KeyPair:**
- `id`, `name`, `fingerprint`
- `public_key`, `created_at`

### Units

- **Memory**: Backend uses **MB**, frontend displays as **GB** (`memory_mb / 1024`)
- **Disk/Volume Size**: Backend uses **GB**, frontend displays directly
- **Dates**: ISO 8601 format, converted to local date format

## Recent Updates

See [CHANGELOG.md](./CHANGELOG.md) for detailed change history.

### Latest Fixes (2024-11-11)

1. ✅ Fixed all JSX render functions to return proper React elements
2. ✅ Fixed TypeScript type errors (`unknown` to `ReactNode`)
3. ✅ Fixed React duplicate key warnings
4. ✅ Corrected all field names to match backend entities
5. ✅ Fixed unit conversions (MB to GB for memory)
6. ✅ Added instance detail page with reset password feature

## License

Part of the JVP project.
