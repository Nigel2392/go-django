const _accordion_controller_name = "accordion";

function initAuditlogsList() {
    if (!window.AdminSite) {
        console.error("AdminSite variable is not defined, cannot continue");
        return;
    }

    var navigation = document.getElementById("navigation");
    if (!navigation) {
        console.error("\"#navigation\" element not found, cannot continue");
        return;
    }

    var auditLogs = document.getElementById("auditlogs");
    if (!auditLogs) {
        console.error("\"#auditlogs\" element not found, cannot continue");
        return;
    }

    var listItems = auditLogs.getElementsByClassName("auditlog-list-item");
    if (!listItems || listItems.length === 0) {
        console.error("\".auditlog-list-item\" elements not found, cannot continue");
        return;
    }

    var cookieName = auditLogs.dataset.cookieName;
    var openedText = window.i18n.gettext("Collapse All");
    var closedText = window.i18n.gettext("Expand All");

    var open = (getCookie(cookieName) === "open");
    var auditLogsExpandButton = document.createElement("button");
    auditLogsExpandButton.type = "button";
    auditLogsExpandButton.className = "auditlogs-expand actions__action";
    auditLogsExpandButton.textContent = open ? openedText : closedText;
    auditLogsExpandButton.addEventListener("click", function() {

        var attr;
        if (open) {
            attr = "close";
            setCookie(cookieName, "closed", 365);
            auditLogsExpandButton.textContent = closedText;
        } else {
            attr = "open";
            setCookie(cookieName, "open", 365);
            auditLogsExpandButton.textContent = openedText;
        }

        var listItems = auditLogs.getElementsByClassName("auditlog-list-item");
        for (var i = 0; i < listItems.length; i++) {
            var item = listItems[i];
            window.AdminSite.stimulusApp.
            getControllerForElementAndIdentifier(item, _accordion_controller_name)[attr]();
        }

        open = !open;
    });

    navigation.insertBefore(
        auditLogsExpandButton, navigation.firstChild,
    );
}

if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initAuditlogsList);
} else {
    initAuditlogsList();
}