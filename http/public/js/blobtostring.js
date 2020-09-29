
// blobbase64tostring converts blob base64 to string
function blobbase64tostring(blobData, callback) {
    let reader = new FileReader();
    reader.onloadend = function () {
        let b64 = reader.result.replace(/^data:.+;base64,/, '');
        callback(b64);
    };
    reader.readAsDataURL(blobData);
}
