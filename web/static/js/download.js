const streamSaver = window.streamSaver;

document.addEventListener("DOMContentLoaded", () => {
    let xhr = new XMLHttpRequest();
    xhr.open("GET", `/d${window.location.pathname}`, true);
    xhr.setRequestHeader('Content-Type', 'application/json');

    xhr.onreadystatechange = () => {
        if (xhr.readyState === 4 && xhr.status === 200) {
            let download = JSON.parse(xhr.responseText);
            handleMetadata(download);
        } else if (xhr.readyState === 4 && xhr.status !== 200) {
            alert(`Error ${xhr.status}: ${xhr.responseText}`);
        }
    };

    xhr.send();
});

const handleMetadata = (download) => {
    // Attempt to decrypt without a password first
    let salt = base64ToArray(download.salt);
    let pepper = location.hash.slice(1);

    deriveKey("", salt, pepper, () => {}, (key, _) => {
        let decryptedName = decryptName(key, download.name);

        if (decryptedName) {
            showDownload(decryptedName, download, key);
        } else {
            promptPassword(download);
        }
    });
}

const showDownload = (name, download, key) => {
    let loading = document.getElementById("loading");
    loading.style.display = "none";

    let passwordPrompt = document.getElementById("password-prompt-div");
    passwordPrompt.style.display = "none";

    let nameSpan = document.getElementById("name");
    nameSpan.textContent = name;

    let expiration = document.getElementById("expiration");
    expiration.textContent = calcTimeRemaining(download.expiration);

    let downloads = document.getElementById("downloads");
    downloads.textContent = download.downloads;

    let size = document.getElementById("size");
    size.textContent = calcFileSize(download.size);

    let downloadDiv = document.getElementById("download-prompt-div");
    downloadDiv.style.display = "inherit";

    let downloadBtn = document.getElementById("download-nopass");
    downloadBtn.addEventListener("click", () => {
        downloadFile(name, download, key);
    })
}

const decryptName = (key, name) => {
    let nameBytes = hexToBytes(name);
    name = decryptString(key, nameBytes);
    return name;
}

const updatePasswordBtn = (txt, disabled) => {
    let btn = document.getElementById("submit");
    btn.value = txt;
    btn.textContent = txt;
    btn.disabled = disabled;
}

const setFormEnabled = on => {
    let fieldset = document.getElementById("download-fieldset");
    fieldset.disabled = !on;
}

const calcTimeRemaining = expiration => {
    let currentTime = new Date();
    let expTime = new Date(expiration);

    let timeDifference = expTime - currentTime;

    const totalSeconds = Math.floor(timeDifference / 1000);
    const totalMinutes = Math.floor(totalSeconds / 60);
    const totalHours = Math.floor(totalMinutes / 60);
    const days = Math.floor(totalHours / 24);

    const hours = totalHours % 24;
    const minutes = totalMinutes % 60;
    const seconds = totalSeconds % 60;

    return `${days} days, ${hours} hours, ${minutes} minutes, ${seconds} seconds`;
}

const calcFileSize = bytes => {
    let thresh = 1000;

    if (Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }

    const units = ['KB', 'MB', 'GB', 'TB'];
    let u = -1;
    const r = 10;

    do {
        bytes /= thresh;
        ++u;
    } while (Math.round(Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


    return bytes.toFixed(1) + ' ' + units[u];
}

const promptPassword = (download) => {
    let loading = document.getElementById("loading");
    loading.style.display = "none";

    let downloadDiv = document.getElementById("password-prompt-div");
    downloadDiv.style.display = "inherit";

    let password = document.getElementById("password");
    let btn = document.getElementById("submit");

    btn.addEventListener("click", () => {
        let salt = base64ToArray(download.salt);
        let pepper = location.hash.slice(1);

        deriveKey(password.value, salt, pepper, () => {
            setFormEnabled(false);
            updatePasswordBtn("Validating", true);
        }, (key, _) => {
            setFormEnabled(true);

            let decryptedName = decryptName(key, download.name);

            if (decryptedName) {
                showDownload(decryptedName, download, key);
            } else {
                updatePasswordBtn("Submit", false);
                alert("Incorrect password");
            }
        });
    });
}

const downloadFile = (name, download, key) => {
    let writer = getFileWriter(name);

    const fetch = (chunkNum) => {
        let xhr = new XMLHttpRequest();
        xhr.open("GET", `/d/${download.id}/${chunkNum}`, true);
        xhr.responseType = 'blob';

        xhr.onreadystatechange = async () => {
            if (xhr.readyState === 4 && xhr.status === 200) {
                let data = new Uint8Array(await xhr.response.arrayBuffer());
                let decryptedChunk = decryptChunk(key, data);
                writer.write(decryptedChunk).then(() => {
                    if (chunkNum === download.chunks) {
                        writer.close().then(r => console.log(r));
                    } else {
                        // Fetch next chunk
                        fetch(chunkNum + 1);
                    }
                });
            } else if (xhr.readyState === 4 && xhr.status !== 200) {
                alert(`Error ${xhr.status}: ${xhr.responseText}`);
            }
        };

        xhr.send();
    }

    // Start with first chunk
    fetch(1);
}

const getFileWriter = (name, length) => {
    // TODO: Need original file size sent to and received from server
    // let fileStream = streamSaver.createWriteStream(name, {
    //     size: length, // (optional filesize) Will show progress
    //     writableStrategy: undefined, // (optional)
    //     readableStrategy: undefined  // (optional)
    // });

    let fileStream = streamSaver.createWriteStream(name);
    return fileStream.getWriter();
}