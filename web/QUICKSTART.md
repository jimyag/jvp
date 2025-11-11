# Quick Start Guide

## Prerequisites

1. Ensure your JVP backend is running on `http://localhost:8080`
2. Node.js 18+ installed

## Run Development Server

```bash
cd web
npm install
npm run dev
```

Visit `http://localhost:3000` to view the dashboard.

## Build for Production

```bash
npm run build
npm start
```

## Features Overview

### 1. Instances Management (`/instances`)
- View all VM instances
- Create new instances with custom CPU, memory, disk
- Start/Stop/Restart instances
- Delete instances
- Monitor instance status

### 2. Volumes Management (`/volumes`)
- Create new storage volumes
- Attach volumes to instances
- Detach volumes from instances
- Delete volumes
- View volume status and attached instances

### 3. Images Management (`/images`)
- View available system images
- Register new images from URLs
- Delete images
- View image details (OS type, size)

### 4. Key Pairs Management (`/keypairs`)
- Create new SSH key pairs (RSA/Ed25519)
- Import existing public keys
- Download private keys
- Delete key pairs
- View key fingerprints

## Design Features

- **Responsive Design**: Works on desktop, tablet, and mobile
- **MotherDuck-inspired UI**: Clean, professional design with:
  - Border-based components
  - Smooth hover effects
  - Clear typography hierarchy
  - Consistent spacing
- **Real-time Updates**: Refresh buttons on all pages
- **Modal Forms**: Easy-to-use forms for creating resources

## Troubleshooting

### API Connection Issues
If you can't connect to the backend:
1. Check that JVP backend is running
2. Verify the backend URL in `next.config.ts`
3. Check browser console for CORS errors

### Build Errors
If you encounter build errors:
```bash
rm -rf node_modules .next
npm install
npm run build
```

## Next Steps

- Customize colors in `tailwind.config.ts`
- Add authentication if needed
- Extend with additional features (snapshots, networking, etc.)
- Deploy to production environment
