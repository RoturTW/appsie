function getCookie(cname) {
  let name = cname + "=";
  let decodedCookie = decodeURIComponent(document.cookie);
  let ca = decodedCookie.split(';');
  for(let i = 0; i < ca.length; i++) {
    let c = ca[i];
    while (c.charAt(0) == ' ') {
      c = c.substring(1);
    }
    if (c.indexOf(name) == 0) {
      return c.substring(name.length, c.length);
    }
  }
  return "";
}

// If session key, check if it's valid
// If no session key, authenticate through rotur and then get session key
window.onload = async function () {
  const session = getCookie("session_id")
  if (session != "") {
    window.location.href = "/"
  } else {
    // Get URL parameters using URLSearchParams
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');

    if (token) {
      const validator = await fetch("https://social.rotur.dev/generate_validator?key=rotur-app-store&auth=" + token)
        .then(v => v.json())
        .then(v => v.validator)
      const auth = await fetch("/api/auth?v=" + encodeURIComponent(validator))
      if (auth.ok) {
        window.location.href = "/"
      }
    } else {
      window.location.href = "https://rotur.dev/auth?return_to=".concat(encodeURIComponent(window.location.href))
    }
  }
};
