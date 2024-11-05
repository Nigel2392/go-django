async function markAsDone(url, csrftoken, update) {
    console.log(url, csrftoken);

    var response = await fetch(url, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrftoken,
        },
    });

    var data = await response.json();
    if (data.status === "success") {
        if (data.done) {
            update("Task marked as done!");
        } else {
            update("Task marked as not done!");
        }
    } else {
        alert("An error occurred");
    }
}

function initForm(form) {
    const formUrl = form.getAttribute("action");
    const csrfTokenInput = form.querySelector(".csrftoken-input");
    const updateElement = form.querySelector(".update");

    form.addEventListener("submit", function(e) {
        e.preventDefault();
        
        markAsDone(formUrl, csrfTokenInput.value, function(message) {
            updateElement.textContent = message;
        });
    });
}

document.addEventListener("DOMContentLoaded", function() {
    const forms = document.querySelectorAll(".todo-form");
    forms.forEach(initForm);
});