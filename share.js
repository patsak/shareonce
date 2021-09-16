
function shareEncrypt() {

    formText = document.getElementById("rawText");
    pass = sjcl.random.randomWords(4,0);
    result = sjcl.encrypt(pass, formText.value);

    body = {
        "cipherText": result,
      };
    response = fetch(window.location.href, {
        method: "POST",
        headers: {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
          },
          body: JSON.stringify(body)
    });
    response.then(data => {
        if (!data.ok) {
            return
        }
        data.json().then(j => {
            input = document.getElementById("input");

            e = document.getElementById("shareLink");

            e.textContent = window.location.href + "l" + "/" + j["id"] + "/" + sjcl.codec.hex.fromBits(pass);
        
            input.remove()
        });
    });

    return false;
}

function show() {
    path = window.location.pathname.split("/");
    out = sjcl.codec.hex.toBits(path[path.length-1]);
    text = document.getElementById("text");

    result = sjcl.decrypt(out, text.textContent);
    text.textContent = result
}


function onLoad() {
    const url = document.getElementById("shareLink");

    url.onclick = function () {
        document.execCommand("copy");
    }

    url.addEventListener("copy", function(event) {
        event.preventDefault();
        if (event.clipboardData) {
            event.clipboardData.setData("text/plain", url.textContent);
            window.alert("copy to clipboard")
        }
    });
}

