document.addEventListener("DOMContentLoaded", function() {
var form = document.querySelector("form");
// Check if form is disabled
if (!form.classList.contains("disabled")) {
	var allBtn = document.querySelectorAll(".m2m-add-all");
	var removeAllBtn = document.querySelectorAll(".m2m-remove-all");
	var options = document.querySelectorAll(".m2m-select option");
	allBtn.forEach(function(btn) {
		btn.addEventListener("click", function() {
			var container = this.parentElement.parentElement;
			var selectors = container.querySelectorAll("select");
			var ownSelect = selectors[0];
			var relatedSelect = selectors[1];
			var options = relatedSelect.options;
			for (var i = 0; i < options.length; i++) {
				var option = document.createElement("option");
				option.value = options[i].value;
				option.text = options[i].text;
				option.selected = true;
				option.addEventListener("dblclick", switchSelected);
				ownSelect.appendChild(option);
			}
			relatedSelect.innerHTML = "";
			setSelected(selectors[0].querySelectorAll("option"));
		});
	});
	removeAllBtn.forEach(function(btn) {
		btn.addEventListener("click", function() {
			var container = this.parentElement.parentElement;
			var selectors = container.querySelectorAll("select");
			var ownSelect = selectors[0];
			var relatedSelect = selectors[1];
			var options = ownSelect.options;
			for (var i = 0; i < options.length; i++) {
				var option = document.createElement("option");
				option.value = options[i].value;
				option.text = options[i].text;
				option.addEventListener("dblclick", switchSelected);
				relatedSelect.appendChild(option);
			}
			ownSelect.innerHTML = "";
			setSelected(selectors[0].querySelectorAll("option"));
		});
	});
	// Add event listener, listen for double click
	options.forEach(function(option) {
		option.addEventListener("dblclick", switchSelected);
	});
	options.forEach(function(option) {
		option.addEventListener("click", function() {
			var container = this.parentElement.parentElement.parentElement;
			var selectors = container.querySelectorAll("select");
			setSelected(selectors[0].querySelectorAll("option"));
		});
	});
	// Switch selected option
	function switchSelected() {
		var container = this.parentElement.parentElement.parentElement;
		var selectors = container.querySelectorAll("select");
		if (this.parentElement === selectors[0]) {
			var ownSelect = selectors[0];
			var relatedSelect = selectors[1];
			var option = document.createElement("option");
			option.value = this.value;
			option.text = this.text;
			relatedSelect.appendChild(option);
			document.removeEventListener(this, switchSelected)
			this.parentElement.removeChild(this);
			option.addEventListener("dblclick", switchSelected)
			setSelected(selectors[0].querySelectorAll("option"));
		} else if (this.parentElement === selectors[1]) {
			var ownSelect = selectors[1];
			var relatedSelect = selectors[0];
			var option = document.createElement("option");
			option.value = this.value;
			option.text = this.text;
			option.selected = true;
			relatedSelect.appendChild(option);
			document.removeEventListener(this, switchSelected)
			this.parentElement.removeChild(this);
			option.addEventListener("dblclick", switchSelected)
			setSelected(selectors[0].querySelectorAll("option"));
		} 
	}
}

function setSelected(options) {
	options.forEach(function(option) {
		option.selected = true;
	});
}
});