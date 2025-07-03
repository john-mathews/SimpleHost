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
        for (const file of fileList) {
            const uploadId = Math.random().toString(36).slice(2) + Date.now();
            const totalChunks = Math.ceil(file.size / CHUNK_SIZE);
            for (let chunkIdx = 0; chunkIdx < totalChunks; chunkIdx++) {
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
                await new Promise((resolve, reject) => {
                    const xhr = new XMLHttpRequest();
                    xhr.open('POST', '/api/upload', true);
                    xhr.onload = function() {
                        if (xhr.status === 200) {
                            uploadedBytes += chunk.size;
                            progressBar.value = Math.round((uploadedBytes / totalBytes) * 100);
                            resolve();
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
