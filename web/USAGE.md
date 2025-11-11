# JVP Admin Dashboard - Usage Guide

## Getting Started

### 1. Start the Dashboard

```bash
cd web
./start.sh
```

Or manually:
```bash
npm install
npm run dev
```

Visit `http://localhost:3000`

### 2. Ensure Backend is Running

The dashboard requires your JVP backend to be running on `http://localhost:8080`.

Start your backend:
```bash
cd ..  # Back to jvp root
go run cmd/main.go
# or
./jvp-server
```

## Using Each Feature

### Managing Instances

#### Create a New Instance
1. Navigate to "Instances" in the sidebar
2. Click "CREATE INSTANCE" button
3. Fill in the form:
   - **Instance Name**: Give it a unique name (e.g., `web-server-01`)
   - **vCPUs**: Number of virtual CPUs (1-32)
   - **Memory**: RAM in MB (e.g., `2048` for 2GB)
   - **Disk Size**: Storage in GB (e.g., `20`)
   - **Image ID**: System image to use (e.g., `ubuntu-22.04`)
   - **Key Pair**: (Optional) SSH key pair name
4. Click "CREATE INSTANCE"

#### Control Instances
- **Start**: Click the play icon (▶) for stopped instances
- **Stop**: Click the stop icon (■) for running instances
- **Restart**: Click the refresh icon (↻) to restart
- **Delete**: Click the trash icon (🗑) to permanently delete

#### View Instance Details
The table shows:
- Name, Status (running/stopped/pending)
- Resources (vCPUs, Memory, Disk)
- IP Address (when available)

---

### Managing Volumes

#### Create a New Volume
1. Navigate to "Volumes" in the sidebar
2. Click "CREATE VOLUME"
3. Fill in the form:
   - **Volume Name**: Unique identifier
   - **Size**: Storage size in GB
   - **Snapshot ID**: (Optional) Create from snapshot
4. Click "CREATE VOLUME"

#### Attach a Volume
1. Find the unattached volume in the table
2. Click the link icon (🔗)
3. Enter the **Instance ID** to attach to
4. Click "ATTACH VOLUME"

#### Detach a Volume
1. Find the attached volume in the table
2. Click the unlink icon (⛓️‍💥)
3. Confirm the action

#### Delete a Volume
1. Ensure the volume is detached
2. Click the trash icon (🗑)
3. Confirm deletion

---

### Managing Images

#### Register a New Image
1. Navigate to "Images" in the sidebar
2. Click "REGISTER IMAGE"
3. Fill in the form:
   - **Image Name**: Descriptive name (e.g., `ubuntu-22.04-server`)
   - **Image URL**: Direct download link to the image file
   - **OS Type**: Select Linux/Windows/Other
   - **Description**: (Optional) Brief description
4. Click "REGISTER IMAGE"

**Note**: The system will download the image from the URL asynchronously.

#### View Image Details
The table displays:
- Name, Status (available/downloading)
- OS Type, File Size
- Creation Date

#### Delete an Image
1. Click the trash icon (🗑) next to the image
2. Confirm deletion
3. **Warning**: Cannot delete images in use by instances

---

### Managing Key Pairs

#### Create a New Key Pair
1. Navigate to "Key Pairs" in the sidebar
2. Click "CREATE KEY PAIR"
3. Fill in the form:
   - **Key Pair Name**: Unique identifier
   - **Key Type**: Choose RSA (2048-bit) or Ed25519
4. Click "CREATE KEY PAIR"
5. **IMPORTANT**: Download the private key immediately
   - This is your only chance to download it
   - Save it securely (e.g., `~/.ssh/my-keypair.pem`)
   - Set permissions: `chmod 600 ~/.ssh/my-keypair.pem`
6. Click "DOWNLOAD PRIVATE KEY"

#### Import an Existing Key Pair
1. Click "IMPORT KEY"
2. Fill in the form:
   - **Key Pair Name**: Unique identifier
   - **Public Key**: Paste your public key content
     - Usually found in `~/.ssh/id_rsa.pub`
     - Format: `ssh-rsa AAAAB3NzaC1yc2E...`
3. Click "IMPORT KEY PAIR"

#### Use Key Pairs with Instances
When creating an instance, specify the key pair name in the "Key Pair Name" field. The public key will be automatically injected into the instance.

#### Connect to Instance with Key Pair
```bash
ssh -i ~/.ssh/my-keypair.pem ubuntu@<instance-ip>
```

#### Delete a Key Pair
1. Click the trash icon (🗑)
2. Confirm deletion
3. **Note**: This only removes the key from JVP, not from existing instances

---

## Common Workflows

### Workflow 1: Create and Launch a VM

1. **Register an image** (if not already available)
   - Go to Images → Register Image
   - Provide Ubuntu 22.04 image URL

2. **Create a key pair** (for SSH access)
   - Go to Key Pairs → Create Key Pair
   - Download and save the private key

3. **Create an instance**
   - Go to Instances → Create Instance
   - Specify CPU, memory, disk
   - Select the image ID
   - Add the key pair name

4. **Start the instance**
   - Click the play icon to start
   - Wait for status to change to "running"

5. **Connect via SSH**
   ```bash
   ssh -i ~/.ssh/keypair.pem ubuntu@<ip-address>
   ```

### Workflow 2: Add Storage to Existing VM

1. **Create a volume**
   - Go to Volumes → Create Volume
   - Specify size (e.g., 50 GB)

2. **Attach to instance**
   - Click the link icon on the volume
   - Enter the instance ID

3. **Mount in the VM** (via SSH):
   ```bash
   # List available disks
   lsblk

   # Format the new disk (e.g., /dev/vdb)
   sudo mkfs.ext4 /dev/vdb

   # Mount it
   sudo mkdir /mnt/data
   sudo mount /dev/vdb /mnt/data
   ```

### Workflow 3: Import Custom Image

1. **Upload image to accessible URL**
   - Use cloud storage (S3, GCS, etc.)
   - Or local HTTP server

2. **Register in JVP**
   - Go to Images → Register Image
   - Paste the image URL
   - Select OS type

3. **Wait for download**
   - Monitor status in Images table
   - Status will change from "pending" to "available"

4. **Use in new instances**
   - Create instance with the new image ID

---

## Tips & Best Practices

### Naming Conventions
- **Instances**: Use descriptive names with purpose and number
  - Good: `web-server-01`, `db-primary`, `cache-redis`
  - Bad: `instance1`, `test`, `vm`

- **Volumes**: Include size and purpose
  - Good: `data-50gb`, `backups-100gb`

- **Images**: Include OS and version
  - Good: `ubuntu-22.04-server`, `debian-11-minimal`

- **Key Pairs**: Use environment or purpose
  - Good: `dev-keypair`, `prod-web-servers`

### Resource Management
- **Stop unused instances** to save resources
- **Detach volumes** before deleting instances
- **Delete unused volumes** to free up storage
- **Remove old images** to save disk space

### Security
- **Never share private keys**
- **Use strong key pair types** (Ed25519 recommended)
- **Limit instance access** with appropriate key pairs
- **Regular backups**: Create volume snapshots before major changes

### Performance
- **Right-size instances**: Don't over-allocate CPU/RAM
- **Monitor status**: Use refresh buttons to check current state
- **Clean up regularly**: Remove unused resources

---

## Troubleshooting

### Problem: Dashboard won't load
**Solution**:
- Check that `npm run dev` is running
- Visit `http://localhost:3000`
- Check browser console for errors

### Problem: API calls fail
**Solution**:
- Ensure JVP backend is running on port 8080
- Check `next.config.ts` for correct backend URL
- Check backend logs for errors

### Problem: Can't create instance
**Solution**:
- Verify image ID exists (check Images page)
- Ensure sufficient resources available
- Check backend logs for detailed error

### Problem: Can't attach volume
**Solution**:
- Verify instance ID is correct
- Ensure volume is not already attached
- Check that instance is running

### Problem: Private key not downloading
**Solution**:
- Check browser's download settings
- Try a different browser
- Ensure popup blockers are disabled

---

## Keyboard Shortcuts

- `Esc`: Close modal dialogs
- `Cmd/Ctrl + R`: Refresh current page

---

## Mobile Usage

The dashboard is fully responsive and works on:
- Tablets (iPad, Android tablets)
- Mobile phones (iOS, Android)

On mobile:
- Tap the hamburger menu (☰) to open navigation
- Swipe to close navigation
- Tables scroll horizontally as needed

---

## Need Help?

- **Documentation**: See `README.md` and `QUICKSTART.md`
- **Project Details**: See `PROJECT_SUMMARY.md`
- **API Issues**: Check JVP backend logs
- **Frontend Issues**: Check browser console

---

**Happy virtualizing! 🚀**
