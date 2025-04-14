import {Endpoints} from "./endpoints.js";
import {
    AdminFileInfoResponse,
    AdminInviteAction,
    AdminUserAction,
    AdminUserInfoResponse
} from "./interfaces.js";

const init = () => {
    setupUserSearch();
    setupFileSearch();
    setupInviteSending();
    setupInviteRevoking();
}

// =============================================================================
// Invite actions
// =============================================================================
const setupInviteSending = () => {
    let inviteList = document.getElementById("invite-emails") as HTMLTextAreaElement;
    if (!inviteList) {
        // Invites not enabled by the server admin
        return;
    }

    let sendInvitesBtn = document.getElementById("send-invites");
    sendInvitesBtn.addEventListener("click", () => {
        let invitesText = inviteList.value.replace(" ", "");
        let invitesList = invitesText.split(",");
        let invites = new AdminInviteAction();
        invites.emails = invitesList;

        fetch(Endpoints.AdminInviteActions.path, {
            method: "POST",
            body: JSON.stringify(invites)
        }).then(async response => {
            if (!response.ok) {
                alert("Error sending invites: " + await response.text());
                return;
            }

            alert("Invite(s) sent!");
            window.location.reload();
        });
    });
}

const setupInviteRevoking = () => {
    let pendingInvites = document.getElementById("pending-invites") as HTMLSelectElement;
    if (!pendingInvites) {
        // Invites not enabled by the server admin
        return;
    }

    let revokeInvitesBtn = document.getElementById("revoke-invites");
    revokeInvitesBtn.addEventListener("click", () => {
        let selectedValues = Array.from(pendingInvites.selectedOptions)
            .map(option => option.value);
        let invites = new AdminInviteAction();
        invites.emails = selectedValues;

        fetch(Endpoints.AdminInviteActions.path, {
            method: "DELETE",
            body: JSON.stringify(invites)
        }).then(async response => {
            if (!response.ok) {
                alert("Failed to revoke pending invites: " + await response.text());
                return;
            }

            alert("Invite(s) revoked!");
            window.location.reload();
        });
    });
}

// =============================================================================
// User admin
// =============================================================================

const setupUserSearch = () => {
    let userSearchBtn = document.getElementById("user-search-btn") as HTMLButtonElement;
    let userIDInput = document.getElementById("user-id") as HTMLInputElement;

    userSearchBtn.addEventListener("click", () => {
        let userID = userIDInput.value;
        fetch(Endpoints.format(Endpoints.AdminUserActions, userID)).then(async response => {
            if (!response.ok) {
                alert("Error fetching user: " + await response.text());
                return;
            }

            let responseDiv = document.getElementById("user-response");
            responseDiv.innerHTML = "";

            let userInfo = new AdminUserInfoResponse(await response.json());
            let userDiv = generateUserActionsHTML(userInfo);
            responseDiv.appendChild(userDiv);
        }).catch((error: Error) => {
            alert("Error fetching user");
            console.error(error);
        })
    });
}

const generateUserActionsHTML = (userInfo: AdminUserInfoResponse): HTMLDivElement => {
    let userResponseDiv = document.createElement("div") as HTMLDivElement;
    userResponseDiv.className = "bordered-box visible";

    let userInfoElement = document.createElement("div");
    userInfoElement.innerHTML = `<div>ID: ${userInfo.id}</div>
<div>Email: ${userInfo.email}</div>
<div>Storage Used (bytes): ${userInfo.storageUsed} / <input id="${userInfo.id}-storage" value="${userInfo.storageAvailable}" type="number"></div>
<div>Send Used (bytes): ${userInfo.sendUsed} / <input id="${userInfo.id}-send" value="${userInfo.sendAvailable}" type="number"></div>`;

    userResponseDiv.appendChild(userInfoElement);
    userResponseDiv.appendChild(document.createElement("br"));

    let deleteButton = document.createElement("button");
    deleteButton.className = "red-button";
    deleteButton.innerText = "Delete User and Uploads";

    let updateButton = document.createElement("button");
    updateButton.className = "accent-btn";
    updateButton.style.marginRight = "5px";
    updateButton.innerText = "Update User Storage";

    userResponseDiv.appendChild(updateButton);
    userResponseDiv.appendChild(deleteButton);

    updateButton.addEventListener("click", () => updateUser(userInfo.id));
    deleteButton.addEventListener("click", () => {
        if (!confirm("Deleting this user will also delete all files they have " +
            "uploaded. Do you wish to proceed?")) {
            return;
        }

        deleteUser(userInfo.id, userResponseDiv);
    });


    if (userInfo.files.length > 0) {
        let header = document.createElement("span");
        header.className = "span-header";
        header.innerText = "Files:"
        userResponseDiv.appendChild(header);
    }

    for (let i = 0; i < userInfo.files.length; i++) {
        let fileDiv = generateFileActionsHTML(userInfo.files[i]);
        userResponseDiv.appendChild(fileDiv);
    }

    return userResponseDiv;
}

const updateUser = (userID: string) => {
    let storageInput = document.getElementById(`${userID}-storage`) as HTMLInputElement;
    let sendInput = document.getElementById(`${userID}-send`) as HTMLInputElement;

    let action = new AdminUserAction();
    action.storageAvailable = storageInput.valueAsNumber;
    action.sendAvailable = sendInput.valueAsNumber;

    if (action.storageAvailable === null || action.sendAvailable === null) {
        console.error("Invalid storage/send value found");
        return;
    }

    fetch(Endpoints.format(Endpoints.AdminUserActions, userID), {
        method: "PUT",
        body: JSON.stringify(action)
    }).then(async response => {
        if (response.ok) {
            alert("User storage/send updated!");
        } else {
            alert("Failed to update user storage/send: " + await response.text());
        }
    });
}

const deleteUser = (userID: string, userDiv: HTMLDivElement) => {
    fetch(Endpoints.format(Endpoints.AdminUserActions, userID), {
        method: "DELETE"
    }).then(async response => {
        if (!response.ok) {
            alert("Failed to delete user! " + await response.text());
        } else {
            alert("User and their content has been deleted!");
            userDiv.innerHTML = "";
            userDiv.className = "hidden";
        }
    }).catch(error => {
        alert("Failed to delete user");
        console.error(error);
    });
}

// =============================================================================
// File admin
// =============================================================================

const setupFileSearch = () => {
    let fileSearchBtn = document.getElementById("file-search-btn") as HTMLButtonElement;
    let fileIDInput = document.getElementById("file-id") as HTMLInputElement;

    fileSearchBtn.addEventListener("click", () => {
        let fileID = fileIDInput.value;
        fetch(Endpoints.format(Endpoints.AdminFileActions, fileID)).then(async response => {
            if (!response.ok) {
                alert("Error fetching file: " + await response.text());
                return
            }

            let responseDiv = document.getElementById("file-response");
            responseDiv.innerHTML = "";

            let fileInfo = new AdminFileInfoResponse(await response.json());
            let fileDiv = generateFileActionsHTML(fileInfo);
            responseDiv.appendChild(fileDiv);
        }).catch(error => {
            console.log(error);
        })
    });
}

const generateFileActionsHTML = (fileInfo: AdminFileInfoResponse): HTMLDivElement => {
    let fileResponseDiv = document.createElement("div") as HTMLDivElement;

    fileResponseDiv.className = "bordered-box visible";

    let fileInfoElement = document.createElement("code");
    fileInfoElement.innerText = `ID: ${fileInfo.id}
Stored Name (encrypted): ${fileInfo.bucketName}
Size: ${fileInfo.size}
Owner ID: ${fileInfo.ownerID}
Modified: ${fileInfo.modified}`;

    fileResponseDiv.appendChild(fileInfoElement);
    fileResponseDiv.appendChild(document.createElement("br"));

    let deleteBtnID = `delete-file-${fileInfo.id}`;
    let deleteButton = document.createElement("button");
    deleteButton.id = deleteBtnID;
    deleteButton.className = "red-button";
    deleteButton.innerText = "Delete File";

    fileResponseDiv.appendChild(deleteButton);

    deleteButton.addEventListener("click", () => {
        if (!confirm("Deleting this file is irreversible. Proceed?")) {
            return;
        }

        fetch(Endpoints.format(Endpoints.AdminFileActions, fileInfo.id), {
            method: "DELETE"
        }).then(async response => {
            if (!response.ok) {
                alert("Failed to delete file! " + await response.text());
            } else {
                alert("The file has been deleted!");
                fileResponseDiv.innerHTML = "";
                fileResponseDiv.className = "hidden";
            }
        }).catch(error => {
            alert("Failed to delete file");
            console.error(error);
        });
    });

    return fileResponseDiv;
}

if (document.readyState !== "loading") {
    init();
} else {
    document.addEventListener("DOMContentLoaded", () => {
        init();
    });
}