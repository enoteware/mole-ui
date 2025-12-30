const { app, BrowserWindow, Tray, Menu, shell } = require('electron');
const { spawn } = require('child_process');
const path = require('path');
const http = require('http');

let mainWindow = null;
let tray = null;
let serverProcess = null;
const PORT = 8081;
const HOST = 'localhost';

// Get the path to the Go binary
function getGoServerPath() {
  const isDev = !app.isPackaged;
  if (isDev) {
    // In development, use the binary from the project root
    return path.join(__dirname, '..', 'bin', 'web-go');
  } else {
    // In production, use the bundled binary
    return path.join(process.resourcesPath, 'bin', 'web-go');
  }
}

// Get the templates directory
function getTemplatesPath() {
  const isDev = !app.isPackaged;
  if (isDev) {
    return path.join(__dirname, '..', 'cmd', 'web', 'templates');
  } else {
    return path.join(process.resourcesPath, 'templates');
  }
}

// Check if server is running
function checkServer(callback) {
  const req = http.request({
    hostname: HOST,
    port: PORT,
    path: '/health',
    method: 'GET',
    timeout: 1000
  }, (res) => {
    callback(res.statusCode === 200);
  });

  req.on('error', () => callback(false));
  req.on('timeout', () => {
    req.destroy();
    callback(false);
  });

  req.end();
}

// Start the Go web server
function startServer() {
  return new Promise((resolve, reject) => {
    const serverPath = getGoServerPath();
    const templatesPath = getTemplatesPath();

    console.log('Starting server:', serverPath);
    console.log('Templates path:', templatesPath);

    // Set environment variables
    const env = {
      ...process.env,
      MOLE_PORT: PORT.toString(),
      MOLE_HOST: HOST,
      MOLE_NO_OPEN: '1',
      MOLE_TEMPLATES: templatesPath
    };

    serverProcess = spawn(serverPath, [], {
      env,
      stdio: ['ignore', 'pipe', 'pipe']
    });

    serverProcess.stdout.on('data', (data) => {
      console.log('Server:', data.toString());
    });

    serverProcess.stderr.on('data', (data) => {
      console.error('Server error:', data.toString());
    });

    serverProcess.on('error', (err) => {
      console.error('Failed to start server:', err);
      reject(err);
    });

    serverProcess.on('exit', (code) => {
      console.log('Server exited with code:', code);
      serverProcess = null;
    });

    // Wait for server to be ready
    let attempts = 0;
    const maxAttempts = 30;

    const checkInterval = setInterval(() => {
      attempts++;
      checkServer((isRunning) => {
        if (isRunning) {
          clearInterval(checkInterval);
          console.log('Server is ready!');
          resolve();
        } else if (attempts >= maxAttempts) {
          clearInterval(checkInterval);
          reject(new Error('Server failed to start in time'));
        }
      });
    }, 500);
  });
}

// Stop the Go web server
function stopServer() {
  if (serverProcess) {
    console.log('Stopping server...');
    serverProcess.kill('SIGTERM');
    serverProcess = null;
  }
}

// Create the main window
function createWindow() {
  const iconPath = path.join(__dirname, 'assets', 'icon.png');
  const windowOptions = {
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    title: 'Mole - Mac System Cleaner',
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    backgroundColor: '#18181b',
    titleBarStyle: 'hiddenInset',
    trafficLightPosition: { x: 16, y: 16 }
  };

  // Add icon only if it exists
  const fs = require('fs');
  if (fs.existsSync(iconPath)) {
    windowOptions.icon = iconPath;
  }

  mainWindow = new BrowserWindow(windowOptions);

  mainWindow.loadURL(`http://${HOST}:${PORT}`);

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Don't close app when window is closed, just hide it
  mainWindow.on('close', (event) => {
    if (!app.isQuitting) {
      event.preventDefault();
      mainWindow.hide();
    }
  });
}

// Create the system tray
function createTray() {
  const fs = require('fs');
  const iconPath = path.join(__dirname, 'assets', 'tray-icon.png');

  // Skip tray if icon doesn't exist
  if (!fs.existsSync(iconPath)) {
    console.log('Tray icon not found, skipping menu bar icon');
    return;
  }

  tray = new Tray(iconPath);

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Open Mole',
      click: () => {
        if (mainWindow) {
          mainWindow.show();
          mainWindow.focus();
        } else {
          createWindow();
        }
      }
    },
    {
      label: 'Open in Browser',
      click: () => {
        shell.openExternal(`http://${HOST}:${PORT}`);
      }
    },
    { type: 'separator' },
    {
      label: 'Server Status',
      enabled: false
    },
    {
      label: serverProcess ? '● Running' : '○ Stopped',
      enabled: false
    },
    { type: 'separator' },
    {
      label: 'Quit Mole',
      click: () => {
        app.isQuitting = true;
        app.quit();
      }
    }
  ]);

  tray.setToolTip('Mole - Mac System Cleaner');
  tray.setContextMenu(contextMenu);

  // Double-click to open window
  tray.on('double-click', () => {
    if (mainWindow) {
      mainWindow.show();
      mainWindow.focus();
    } else {
      createWindow();
    }
  });
}

// App lifecycle
app.whenReady().then(async () => {
  try {
    // Start the Go server first
    await startServer();

    // Create the main window
    createWindow();

    // Create the system tray
    createTray();

    console.log('Mole app is ready!');
  } catch (err) {
    console.error('Failed to start Mole:', err);
    app.quit();
  }
});

app.on('window-all-closed', () => {
  // Don't quit on macOS when all windows are closed
  // The app stays running in the menu bar
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  } else {
    mainWindow.show();
  }
});

app.on('before-quit', () => {
  app.isQuitting = true;
});

app.on('will-quit', () => {
  stopServer();
});

// Handle uncaught exceptions
process.on('uncaughtException', (err) => {
  console.error('Uncaught exception:', err);
});
