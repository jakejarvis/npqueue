document.addEventListener("DOMContentLoaded", updateData, false);

async function fetchData() {
  let request = await fetch("https://hidden-depths-42970.herokuapp.com/api/list");
  if(request.status == 200) {
    console.log("New JSON successfully fetched!");
    let data = await request.json();
    return data;
  }
  throw new Error(request.status);
}

function updateData() {
  fetchData().then(data => updateTable(data)).catch(data => console.log(data));

  // Re-fetch data every 1 minute
  setTimeout(updateData, 60000);
}

function updateTable(data) {
  active = document.getElementById("active");
  active.innerText = `Active Players: ${data.currentPlayers} / 60`;

  queue = document.getElementById("queue");
  queue.innerText = `In Queue: ${data.currentQueue}`;

  table = document.getElementsByTagName("table")[0].getElementsByTagName("tbody")[0];
  table.innerHTML = "";


  data.players = data.players.sort((a, b) => (a.id > b.id) ? 1 : ((b.id > a.id) ? -1 : 0));

  data.players.forEach(function(player, i) {
    newRow = table.insertRow(table.rows.length);

    newCell = newRow.insertCell(0);
    newText = document.createTextNode(`${i+1}`)
    newCell.appendChild(newText);

    newCell = newRow.insertCell(1);
    newText = document.createTextNode(`${player.id}`)
    newCell.appendChild(newText);

    newCell = newRow.insertCell(2);
    a = document.createElement('a');
    a.setAttribute('href', `https://www.twitch.tv/${player.identifiers[3]}`);
    a.setAttribute('target', '_blank');
    a.setAttribute('rel', 'noopener noreferrer nofollow');
    a.innerHTML = player.identifiers[3];
    newCell.appendChild(a);

    newCell = newRow.insertCell(3);
    a = document.createElement('a');
    a.setAttribute('href', `https://www.nopixel.net/upload/index.php?members/${player.identifiers[4]}`);
    a.setAttribute('target', '_blank');
    a.setAttribute('rel', 'noopener noreferrer nofollow');
    a.innerHTML = player.identifiers[0];
    newCell.appendChild(a);

    newCell = newRow.insertCell(4);
    a = document.createElement('a');
    a.setAttribute('href', `https://steamcommunity.com/profiles/${player.identifiers[2]}`);
    a.setAttribute('target', '_blank');
    a.setAttribute('rel', 'noopener noreferrer nofollow');
    a.innerHTML = player.name;
    newCell.appendChild(a);

    newCell = newRow.insertCell(5);
    newText = document.createTextNode(`${player.identifiers[1]}`)
    newCell.appendChild(newText);

    newCell = newRow.insertCell(6);
    newText = document.createTextNode(`${player.identifiers[2]}`)
    newCell.appendChild(newText);

    newCell = newRow.insertCell(7);
    newText = document.createTextNode(`${player.ping} ms`)
    newCell.appendChild(newText);
  });
}