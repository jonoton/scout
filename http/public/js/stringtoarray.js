
// base64stringtobinaryuint8array converts base64 string to binary uint8array
function base64stringtobinaryuint8array(b64) {
    let strData = atob(b64);
    let charData = strData.split('').map(function (x) { return x.charCodeAt(0); });
    let binaryuint8array = new Uint8Array(charData);
    return binaryuint8array;
}
