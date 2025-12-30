// Preload script for Mole Electron app
// This runs in the renderer process before web content loads

const { contextBridge } = require('electron');

// Expose safe APIs to the renderer if needed in the future
contextBridge.exposeInMainWorld('mole', {
  version: '1.0.0'
});
