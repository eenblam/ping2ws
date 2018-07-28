function getOrAddElementById(id) {
  let e = document.getElementById(id);
  if (e === null) {
    // Create new element and append to #update-container
    let list = document.getElementById("update-list");
    let e = document.createElement("li");
    e.id = id;
    list.appendChild(e);
  }
  return e;
}

function parseUpdate(update) {
  let data = update.data;
  let parsed = JSON.parse(data);
  return parsed;
};

function handleUpdate(update) {
  let id = `target-${update.target}`;
  let elt = getOrAddElementById(id);
  let targetStatus = update.up ? "UP" : "DOWN";
  elt.innerHTML = `${update.target} is ${targetStatus}`;
}

let portString = location.port ? `:${location.port}` : '';
let loc = `wss://${location.hostname}${portString}/monitor`;
console.log(loc);

const ws = new WebSocket(loc);
ws.onmessage = (update) => {
  update = parseUpdate(update);
  handleUpdate(update);
};
