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
    uploadFiles(fileList, status, progressBar) {
        const formData = new FormData();
        for (const file of fileList) {
            formData.append('file', file, file.webkitRelativePath || file.name);
        }
        // Add current folder id from URL as folder_id
        const params = new URLSearchParams(window.location.search);
        const folderId = params.get('folderId') || 'root';
        formData.append('folder_id', folderId);

        // Use XMLHttpRequest for progress
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/upload', true);
        xhr.upload.onprogress = function(e) {
            if (e.lengthComputable) {
                progressBar.style.display = '';
                progressBar.value = (e.loaded / e.total) * 100;
            }
        };
        xhr.onload = function() {
            progressBar.style.display = 'none';
            progressBar.value = 0;
            status.textContent = xhr.responseText;
            // Auto-refresh file list after upload
            const filesList = window.parent ? window.parent.document.getElementById('files-list') : document.getElementById('files-list');
            if (filesList) {
                // Use HTMX if available
                if (window.parent && window.parent.htmx) {
                    window.parent.htmx.trigger(filesList, 'refresh');
                } else if (window.htmx) {
                    window.htmx.trigger(filesList, 'refresh');
                } else {
                    // Fallback: reload the page
                    window.location.reload();
                }
            }
        };
        xhr.onerror = function() {
            progressBar.style.display = 'none';
            progressBar.value = 0;
            status.textContent = 'Upload failed.';
        };
        progressBar.value = 0;
        progressBar.style.display = '';
        status.textContent = 'Uploading...';
        xhr.send(formData);
    }
}
customElements.define('file-uploader', FileUploader);
