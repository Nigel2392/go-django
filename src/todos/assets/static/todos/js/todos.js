async function markAsDone(url, csrftoken) {
    var response = await fetch(url, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            "X-CSRFToken": csrftoken,
        },
    });

    var data = await response.json();
    if (data.status === "success") {
        alert("Todo marked as done");
    } else {
        alert("An error occurred");
    }
}

function initForm(form) {
    const formUrl = form.getAttribute("action");
    const csrfTokenInput = form.querySelector(".csrftoken-input");

    form.addEventListener("submit", function(e) {
        e.preventDefault();
        markAsDone(formUrl, csrfTokenInput.value);
    });
}

document.addEventListener("DOMContentLoaded", function() {
    const forms = document.querySelectorAll(".todo-form");
    forms.forEach(initForm);
});