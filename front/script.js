const getData = async (path, params) => {
  const resp = await fetch(path, params);
  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(text);
  }
  const data = await resp.blob();
  return data;
};

const form = document.getElementById("settings");
form.addEventListener("input", () => renew());

let changed = 0;
const renew = () => {
  changed++;
  const current = changed;
  this.setTimeout(() => {
    if (current === changed) {
      changed = 0;
      fetchData();
    }
  }, 1000);
};

function getFormData() {
  const formdata = new FormData(form);
  return formdata;
}

const fetchData = () => {
  error.classList.add("hide");
  getData("http://localhost:80/draw", { method: "POST", body: getFormData() })
    .then((data) => {
      const formFields = Object.fromEntries(new FormData(form).entries());
      const png = URL.createObjectURL(data);
      view.setAttribute("src", png);
      view.style.backgroundColor = formFields["background"];
      view.style.borderColor = formFields["background"];
      wrapper.classList.remove("hide");
      if (formFields["func"] === "saddle") {
        despair.style.opacity = "25%";
      } else {
        despair.style.opacity = "0%";
      }
      settings.classList.remove("hide");
    })
    .catch((err) => {
      wrapper.classList.add("hide");
      settings.classList.add("hide");
      console.error(err);
      error.textContent = err;
      error.classList.remove("hide");
    });
};

// init
fetchData();
