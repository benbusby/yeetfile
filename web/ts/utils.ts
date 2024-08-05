const chunkSize: number = 10000000;

enum ExpUnits {
    Minutes = 1,
    Hours,
    Days
}

/**
 * Converts a hex string to a Uint8Array
 * @param {string} hexString - The hex string to convert to an array
 * @returns {Uint8Array} The byte array representation of the hex string
 */
const hexToBytes = (hexString: string): Uint8Array =>
    Uint8Array.from(hexString.match(/.{1,2}/g).map((byte: string) =>
        parseInt(byte, 16)));

/**
 * Converts a Uint8Array of bytes to a hexadecimal string
 * @param {Uint8Array} bytes - The bytes to convert to a hex string
 * @returns {string} The hex string representation of the array
 */
const toHexString = (bytes: Uint8Array): string =>
    bytes.reduce((str: string, byte: number) =>
        str + byte.toString(16).padStart(2, '0'), '');

/**
 * Returns the total number of chunks required to upload
 * a file to YeetFile
 * @param {number} size - The file size, in bytes
 * @returns {number} The number of chunks required to upload the file
 */
const getNumChunks = (size: number): number =>
    Math.ceil(size / chunkSize);

/**
 * Generates a random string of `size` characters
 * @param {number} size - The desired length of the output random string
 * @returns {string} The random string of `size` characters
 */
const genRandomString = (size: number): string =>
    (Math.random().toString(36)+'00000000000000000').slice(2, size+2)

/**
 * Concatenates two Uint8Array elements together
 * @param {Uint8Array} a - The beginning of the desired concatenated output
 * @param {Uint8Array} b - The end of the desired concatenated output
 * @returns {Uint8Array} An array containing `a` elements, then `b` elements
 */
const concatTypedArrays = (a: Uint8Array, b: Uint8Array): Uint8Array => {
    let c = new Uint8Array(a.length + b.length);
    c.set(a, 0);
    c.set(b, a.length);
    return c;
}

/**
 * Generates a human-readable file size from a number of bytes
 * @param {number} bytes - The bytes to convert to a readable string
 * @returns {string} A readable file size string (i.e. 12KB, 2.4MB, etc)
 */
const calcFileSize = (bytes: number): string => {
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
    } while (Math.round(
        Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


    return bytes.toFixed(1) + ' ' + units[u];
}

/**
 * Returns a string representation of a file's intended expiration
 * @param exp {number} - The numeric portion of a file's expiration
 * @param units {ExpUnits} - The expiration units (Minutes, Hours, Days)
 * @returns {string} An expiration string (i.e. "5m" for 5 minutes,
 *      "10d" for 10 days, etc)
 */
const getExpString = (exp: number, units: ExpUnits): string => {
    switch (units) {
        case ExpUnits.Minutes:
            return `${exp}m`
        case ExpUnits.Hours:
            return `${exp}h`
        case ExpUnits.Days:
            return `${exp}d`
    }
}

/**
 * Converts a base64 string into a Uint8Array
 * @param base64 {string} - The base64 string to convert
 * @returns {Uint8Array} The Uint8Array derived from the base64 string
 */
const base64ToArray = (base64: string): Uint8Array => {
    let binaryString = atob(base64);
    let bytes = new Uint8Array(binaryString.length);
    for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
    }

    return bytes;
}

/**
 * Formats a date in the local timezone with a language-sensitive representation
 * @param date {string} - The string date
 * @returns {string} The formatted date string
 */
const formatDate = (date: string): string => {
    let localDate = new Date(date);
    return localDate.toLocaleString();
}

/**
 * Validates the provided expiration value for a file
 * @param {number} exp - The numeric portion of the specified expiration
 * @param {ExpUnit} unit - The unit of the expiration
 * @returns {boolean} True if the expiration value is valid, else false
 */
const validateExpiration = (exp: number, unit: ExpUnits): boolean => {
    let maxDays = 30;
    let maxHours = 24 * maxDays;
    let maxMinutes = 60 * maxHours;

    if (unit === ExpUnits.Minutes) {
        if (exp <= 0 || exp > maxMinutes) {
            alert(`Expiration must be between 0-${maxMinutes} minutes`);
            return false;
        }
    }

    if (unit === ExpUnits.Hours) {
        if (exp <= 0 || exp > maxHours) {
            alert(`Expiration must be between 0-${maxHours} hours`);
            return false;
        }
    }

    if (unit === ExpUnits.Days) {
        if (exp <= 0 || exp > maxDays) {
            alert(`Expiration must be between 0-${maxDays} days`);
            return false;
        }
    }

    return true;
}

/**
 * Converts the selected index of the units dropdown to an ExpUnits enum value
 * @param index {number} - The selected index
 * @returns {ExpUnits} - The ExpUnits enum value matching the index
 */
const indexToExpUnit = (index: number): ExpUnits => {
    switch (index) {
        case 1:
            return ExpUnits.Minutes;
        case 2:
            return ExpUnits.Hours;
        case 3:
            return ExpUnits.Days;
        default:
            return ExpUnits.Minutes;
    }
}

/**
 * Update an html button element with a new label or by enabling/disabling it.
 * @param btn {HTMLButtonElement} - The button element to modify
 * @param disabled {boolean} - True to disable the button, false to keep it enabled
 * @param label {string} - The label to use for the provided button
 * @returns {void}
 */
const updateButton = (
    btn: HTMLButtonElement,
    disabled: boolean,
    label: string,
): void => {
    btn.disabled = disabled;
    btn.innerText = label;
}