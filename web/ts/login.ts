import * as crypto from "./crypto.js";
import * as localstorage from "./localstorage.js";
import { Endpoints } from "./endpoints.js";
import { Login, LoginResponse } from "./interfaces.js";

let vaultPasswordDialog;
let twoFactorDialog;
let vaultPasswordCB;
let loginBtn;
let buttonLabel;

const init = () => {
    vaultPasswordCB = document.getElementById("vault-pass-cb");
    vaultPasswordDialog = document.getElementById("vault-pass-dialog");
    twoFactorDialog = document.getElementById("two-factor-dialog");
    loginBtn = document.getElementById("login-btn");
    buttonLabel = loginBtn.value;
    loginBtn.addEventListener("click", async () => {
        await login("");
    });

    let forgotLink = document.getElementById("forgot-password") as HTMLAnchorElement;
    forgotLink.addEventListener("click", event => {
        event.preventDefault();
        let identifier = document.getElementById("identifier") as HTMLInputElement;
        if (identifier.value.indexOf("@") > 0) {
            window.location.assign(forgotLink.href + `?email=${identifier.value}`);
        } else {
            window.location.assign(forgotLink.href);
        }
    });

    vaultPasswordCB.checked = localstorage.getVaultPasswordSetting();

    // Enter key submits login form
    document.addEventListener("keydown", (event: KeyboardEvent) => {
        if (event.key === "Enter") {
            loginBtn.click();
        }
    });
}

const resetLoginButton = () => {
    let btn = document.getElementById("login-btn") as HTMLButtonElement;
    btn.disabled = false;
    btn.value = buttonLabel;
}

const disableInputs = (disabled: boolean) => {
    let elements = [
        document.getElementById("login-btn") as HTMLButtonElement,
        document.getElementById("identifier") as HTMLInputElement,
        document.getElementById("password") as HTMLInputElement
    ];

    for (let i in elements) {
        elements[i].disabled = disabled;
    }

    let spinner = document.getElementById("login-spinner");
    let forgotPw = document.getElementById("forgot-password");

    spinner.style.display = disabled ? "inline" : "none";
    forgotPw.style.display = disabled ? "none" : "inline";
}

const login = async (twoFactorCode: string) => {
    disableInputs(true);

    let identifier = document.getElementById("identifier") as HTMLInputElement;
    let password = document.getElementById("password") as HTMLInputElement;

    if (!isValidIdentifier(identifier.value)) {
        return;
    }

    let userKey = await crypto.generateUserKey(identifier.value, password.value);
    let loginKeyHash = await crypto.generateLoginKeyHash(userKey, password.value);

    let url = new URL(window.location.href);
    let params = new URLSearchParams(url.search);
    let next = Endpoints.HTMLAccount.path;
    if (params.get("next")) {
        next = params.get("next")
    }

    let loginBody = new Login();
    loginBody.loginKeyHash = loginKeyHash;
    loginBody.identifier = identifier.value;
    loginBody.code = twoFactorCode;

    fetch(Endpoints.Login.path, {
        method: "POST",
        body: JSON.stringify(loginBody, jsonReplacer)
    }).then(async response => {
        if (!response.ok) {
            if (response.status == 403) {
                showTwoFactorDialog();
            } else {
                let errMsg = await response.text();
                showMessage(`Error ${response.status}: ${errMsg}`, true);
                disableInputs(false);
            }
        } else {
            let loginResponse = new LoginResponse(await response.json());
            let privKey = new Uint8Array(await crypto.decryptChunk(
                userKey, loginResponse.protectedKey));
            let pubKey = loginResponse.publicKey;

            if (vaultPasswordCB.checked) {
                showVaultPassDialog(privKey, pubKey);
            } else {
                localstorage.disableVaultPasswordSetting()
                const dbModule = await import('./db.js');
                let db = new dbModule.YeetFileDB();
                db.insertVaultKeyPair(privKey, pubKey, "", success => {
                    if (success) {
                        window.location.assign(next);
                    } else {
                        alert("Failed to insert vault keys into indexeddb");
                        window.location.assign(Endpoints.Logout.path);
                    }
                });
            }
        }
    });
}

const isValidIdentifier = (identifier) => {
    if (identifier.includes("@")) {
        return true;
    } else {
        if (identifier.length !== 16) {
            showMessage("Missing email or 16-digit account ID", true);
            disableInputs(false);
            return false;
        }

        return true;
    }
}

const showTwoFactorDialog = () => {
    let dialog = document.getElementById("two-factor-dialog") as HTMLDialogElement;
    let codeInput = document.getElementById("two-factor-code") as HTMLInputElement;
    let submit = document.getElementById("submit-2fa") as HTMLButtonElement;
    let cancel = document.getElementById("cancel-2fa") as HTMLButtonElement;

    codeInput.addEventListener("keydown", (event: KeyboardEvent) => {
        if (event.key === "Enter") {
            submit.click();
        }
    });

    submit.addEventListener("click",  () => {
        dialog.close();
        login(codeInput.value);
    });

    cancel.addEventListener("click", () => {
        disableInputs(false);
        dialog.close();
    });

    dialog.showModal();
}

const showVaultPassDialog = async (
    privKeyBytes: Uint8Array,
    pubKeyBytes: Uint8Array,
) => {
    const dbModule = await import('./db.js');
    let db = new dbModule.YeetFileDB();

    let cancel = document.getElementById("cancel-pass")
    cancel.addEventListener("click", async () => {
        resetLoginButton();
        await db.removeKeys(success => {
            if (success) {
                fetch(Endpoints.Logout.path).catch(() => {
                    console.warn("error logging user out");
                });
            } else {
                console.warn("error removing keys");
            }
        });
        vaultPasswordDialog.close();
    });

    let submit = document.getElementById("submit-pass");
    submit.addEventListener("click", async () => {
        localstorage.enableVaultPasswordSetting();
        let passwordInput = document.getElementById("vault-pass") as HTMLInputElement;
        let password = passwordInput.value;
        await db.insertVaultKeyPair(
            privKeyBytes,
            pubKeyBytes,
            password,
            success => {
                if (success) {
                    window.location.assign(Endpoints.HTMLAccount.path);
                } else {
                    alert("Failed to insert keys into indexeddb");
                }
            });
    });

    vaultPasswordDialog.showModal();
}

if (document.readyState !== 'loading') {
    init();
} else {
    document.addEventListener('DOMContentLoaded', () => {
        init();
    });
}
