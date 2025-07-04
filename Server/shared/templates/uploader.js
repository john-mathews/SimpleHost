class FileUploader extends HTMLElement {
    constructor() {
        super();
        this.attachShadow({ mode: 'open' });
        this.shadowRoot.innerHTML = `
            <style>
                .dropzone {
                    border: 2px dashed #888;
                    border-radius: 8px;
                    padding: 2em;
                    text-align: center;
                    background: #fafafa;
                    cursor: pointer;
                    transition: background 0.2s;
                }
                .dropzone.dragover {
                    background: #e0e0e0;
                }
                .browse-btns {
                    margin-top: 1em;
                }
                .browse-btns button {
                    margin: 0 0.5em;
                }
                progress {
                    width: 100%;
                    margin-top: 1em;
                    height: 1.2em;
                }
            </style>
            <div class="dropzone" tabindex="0">
                <p>Drag & Drop files here, or</p>
                <div class="browse-btns">
                    <button type="button" id="browseFilesBtn">Select Files</button>
                    <button type="button" id="browseFolderBtn">Select Folder</button>
                </div>
                <input type="file" id="fileInput" style="display:none" multiple />
                <input type="file" id="folderInput" style="display:none" multiple webkitdirectory />
            </div>
            <progress id="progressBar" value="0" max="100" style="display:none"></progress>
            <div id="status"></div>
        `;
    }
    connectedCallback() {
        const dropzone = this.shadowRoot.querySelector('.dropzone');
        const fileInput = this.shadowRoot.getElementById('fileInput');
        const folderInput = this.shadowRoot.getElementById('folderInput');
        const browseFilesBtn = this.shadowRoot.getElementById('browseFilesBtn');
        const browseFolderBtn = this.shadowRoot.getElementById('browseFolderBtn');
        const status = this.shadowRoot.getElementById('status');
        const progressBar = this.shadowRoot.getElementById('progressBar');

        browseFilesBtn.addEventListener('click', (e) => {
            e.stopImmediatePropagation();
            fileInput.click();
        });
        browseFolderBtn.addEventListener('click', (e) => {
            e.stopImmediatePropagation();
            folderInput.click();
        });
        dropzone.addEventListener('click', e => {
            // Only trigger file input if the dropzone itself (not a child) is clicked
            if (e.target === dropzone) {
                fileInput.click();
            }
        });

        fileInput.addEventListener('change', e => {
            this.uploadFiles(fileInput.files, status, progressBar);
        });
        folderInput.addEventListener('change', e => {
            this.uploadFiles(folderInput.files, status, progressBar);
        });

        dropzone.addEventListener('dragover', e => {
            e.preventDefault();
            dropzone.classList.add('dragover');
        });
        dropzone.addEventListener('dragleave', e => {
            dropzone.classList.remove('dragover');
        });
        dropzone.addEventListener('drop', e => {
            e.preventDefault();
            dropzone.classList.remove('dragover');
            this.uploadFiles(e.dataTransfer.files, status, progressBar);
        });
    }
    async uploadFiles(fileList, status, progressBar) {
        const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB
        const params = new URLSearchParams(window.location.search);
        const folderId = params.get('folderId') || 'root';
        let totalBytes = 0;
        let uploadedBytes = 0;
        for (const file of fileList) {
            totalBytes += file.size;
        }
        progressBar.value = 0;
        progressBar.max = 100;
        progressBar.style.display = '';
        status.textContent = 'Uploading...';
        // Track bulk conflict choice
        let conflictChoice = null; // 'overwrite', 'skip', 'overwriteAll', 'skipAll'
        for (const file of fileList) {
            const uploadId = Math.random().toString(36).slice(2) + Date.now();
            const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
            let skipFile = false;
            let overwrite = false;
            for (let chunkIdx = 0; chunkIdx < totalChunks; chunkIdx++) {
                if (skipFile) break;
                const start = chunkIdx * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const chunk = file.slice(start, end);
                const formData = new FormData();
                formData.append('chunk', chunk);
                formData.append('file_name', file.webkitRelativePath || file.name);
                formData.append('upload_id', uploadId);
                formData.append('chunk_index', chunkIdx);
                formData.append('total_chunks', totalChunks);
                formData.append('folder_id', folderId);
                if (overwrite || conflictChoice === 'overwriteAll') formData.append('overwrite', 'true');
                await new Promise((resolve, reject) => {
                    const xhr = new XMLHttpRequest();
                    xhr.open('POST', '/api/upload', true);
                    xhr.onload = async function() {
                        if (xhr.status === 200) {
                            uploadedBytes += chunk.size;
                            progressBar.value = Math.round((uploadedBytes / totalBytes) * 100);
                            resolve();
                        } else if (xhr.status === 409 && xhr.responseText === 'EXISTS') {
                            if (chunkIdx === 0) {
                                if (conflictChoice === 'overwriteAll') {
                                    overwrite = true;
                                    formData.set('overwrite', 'true');
                                    const retryXhr = new XMLHttpRequest();
                                    retryXhr.open('POST', '/api/upload', true);
                                    retryXhr.onload = function() {
                                        if (retryXhr.status === 200) {
                                            uploadedBytes += chunk.size;
                                            progressBar.value = Math.round((uploadedBytes / totalBytes) * 100);
                                            resolve();
                                        } else {
                                            status.textContent = 'Upload failed.';
                                            progressBar.style.display = 'none';
                                            reject(retryXhr.responseText);
                                        }
                                    };
                                    retryXhr.onerror = function() {
                                        status.textContent = 'Upload failed.';
                                        progressBar.style.display = 'none';
                                        reject('Network error');
                                    };
                                    retryXhr.send(formData);
                                } else if (conflictChoice === 'skipAll') {
                                    skipFile = true;
                                    resolve();
                                } else {
                                    // Show custom dialog for this file
                                    const userChoice = await window.showFileConflictDialogBulk?.(file.name) || await window.showFileConflictDialog(file.name);
                                    if (userChoice === 'overwrite') {
                                        overwrite = true;
                                        formData.set('overwrite', 'true');
                                        const retryXhr = new XMLHttpRequest();
                                        retryXhr.open('POST', '/api/upload', true);
                                        retryXhr.onload = function() {
                                            if (retryXhr.status === 200) {
                                                uploadedBytes += chunk.size;
                                                progressBar.value = Math.round((uploadedBytes / totalBytes) * 100);
                                                resolve();
                                            } else {
                                                status.textContent = 'Upload failed.';
                                                progressBar.style.display = 'none';
                                                reject(retryXhr.responseText);
                                            }
                                        };
                                        retryXhr.onerror = function() {
                                            status.textContent = 'Upload failed.';
                                            progressBar.style.display = 'none';
                                            reject('Network error');
                                        };
                                        retryXhr.send(formData);
                                    } else if (userChoice === 'overwriteAll') {
                                        conflictChoice = 'overwriteAll';
                                        overwrite = true;
                                        formData.set('overwrite', 'true');
                                        const retryXhr = new XMLHttpRequest();
                                        retryXhr.open('POST', '/api/upload', true);
                                        retryXhr.onload = function() {
                                            if (retryXhr.status === 200) {
                                                uploadedBytes += chunk.size;
                                                progressBar.value = Math.round((uploadedBytes / totalBytes) * 100);
                                                resolve();
                                            } else {
                                                status.textContent = 'Upload failed.';
                                                progressBar.style.display = 'none';
                                                reject(retryXhr.responseText);
                                            }
                                        };
                                        retryXhr.onerror = function() {
                                            status.textContent = 'Upload failed.';
                                            progressBar.style.display = 'none';
                                            reject('Network error');
                                        };
                                        retryXhr.send(formData);
                                    } else if (userChoice === 'skip') {
                                        skipFile = true;
                                        resolve();
                                    } else if (userChoice === 'skipAll') {
                                        conflictChoice = 'skipAll';
                                        skipFile = true;
                                        resolve();
                                    } else {
                                        // Default: skip
                                        skipFile = true;
                                        resolve();
                                    }
                                }
                            } else {
                                // Should not happen, but skip if so
                                skipFile = true;
                                resolve();
                            }
                        } else {
                            status.textContent = 'Upload failed.';
                            progressBar.style.display = 'none';
                            reject(xhr.responseText);
                        }
                    };
                    xhr.onerror = function() {
                        status.textContent = 'Upload failed.';
                        progressBar.style.display = 'none';
                        reject('Network error');
                    };
                    xhr.send(formData);
                });
            }
        }
        progressBar.style.display = 'none';
        progressBar.value = 0;
        status.textContent = 'Upload complete.';
        // Auto-refresh file list after upload
        const filesList = window.parent ? window.parent.document.getElementById('files-list') : document.getElementById('files-list');
        if (filesList) {
            if (window.parent && window.parent.htmx) {
                window.parent.htmx.trigger(filesList, 'refresh');
            } else if (window.htmx) {
                window.htmx.trigger(filesList, 'refresh');
            } else {
                window.location.reload();
            }
        }
    }
}
customElements.define('file-uploader', FileUploader);
// Helper for custom dialog (can be replaced with a better UI)
window.showFileConflictDialogBulk = async function(filename) {
    return new Promise((resolve) => {
        // Create modal
        let modal = document.createElement('div');
        modal.style.position = 'fixed';
        modal.style.top = '0';
        modal.style.left = '0';
        modal.style.width = '100vw';
        modal.style.height = '100vh';
        modal.style.background = 'rgba(0,0,0,0.4)';
        modal.style.display = 'flex';
        modal.style.alignItems = 'center';
        modal.style.justifyContent = 'center';
        modal.style.zIndex = '9999';
        modal.innerHTML = `
            <div style="background:#fff;padding:2em;border-radius:8px;max-width:90vw;text-align:center;box-shadow:0 2px 12px #0002;">
                <div style="margin-bottom:1em;font-size:1.1em;">File '<b>${filename}</b>' already exists.<br>What do you want to do?</div>
                <button id="owBtn">Overwrite</button>
                <button id="skBtn">Skip</button>
                <button id="owAllBtn">Overwrite All</button>
                <button id="skAllBtn">Skip All</button>
            </div>
        `;
        document.body.appendChild(modal);
        modal.querySelector('#owBtn').onclick = () => { document.body.removeChild(modal); resolve('overwrite'); };
        modal.querySelector('#skBtn').onclick = () => { document.body.removeChild(modal); resolve('skip'); };
        modal.querySelector('#owAllBtn').onclick = () => { document.body.removeChild(modal); resolve('overwriteAll'); };
        modal.querySelector('#skAllBtn').onclick = () => { document.body.removeChild(modal); resolve('skipAll'); };
    });
};
