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
            </style>
            <div class="dropzone" tabindex="0">
                <p>Drag & Drop files here, or <button type="button" id="browseBtn">Browse</button></p>
                <input type="file" id="fileInput" style="display:none" multiple />
            </div>
            <div id="status"></div>
        `;
    }
    connectedCallback() {
        const dropzone = this.shadowRoot.querySelector('.dropzone');
        const fileInput = this.shadowRoot.getElementById('fileInput');
        const browseBtn = this.shadowRoot.getElementById('browseBtn');
        const status = this.shadowRoot.getElementById('status');

        browseBtn.addEventListener('click', () => fileInput.click());
        dropzone.addEventListener('click', e => {
            if (e.target !== browseBtn) fileInput.click();
        });

        fileInput.addEventListener('change', e => {
            this.uploadFiles(fileInput.files, status);
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
            this.uploadFiles(e.dataTransfer.files, status);
        });
    }
    uploadFiles(fileList, status) {
        const formData = new FormData();
        for (const file of fileList) {
            formData.append('file', file, file.name);
        }
        // Add current folder id from URL as folder_id
        const params = new URLSearchParams(window.location.search);
        const folderId = params.get('folderId') || 'root';
        formData.append('folder_id', folderId);
        fetch('/api/upload', {
            method: 'POST',
            body: formData
        }).then(r => r.text()).then(msg => {
            status.textContent = msg;
        }).catch(err => {
            status.textContent = 'Upload failed.';
        });
    }
}
customElements.define('file-uploader', FileUploader);
