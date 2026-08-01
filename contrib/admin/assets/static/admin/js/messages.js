const MESSAGE_COOLDOWN_SECONDS = 5; // seconds

document.addEventListener('DOMContentLoaded', function () {
    let messages = document.querySelectorAll(".messages .message");
    let messageAnimation = [
        { opacity: 1, height: "100%"},
        { opacity: 0, height: "0%"}
    ];
    let messageAnimationOptions = {
        duration: 200,
        easing: 'ease-in-out'
    };
    messages.forEach(function(elem){
        elem.style.cursor = "pointer";
        elem.addEventListener("click", function(){
            let anim = elem.animate(messageAnimation, messageAnimationOptions)
            anim.onfinish = () => {
                elem.remove();
            };
        });
    });
    setTimeout(function(){
        for (let i = 0; i < messages.length; i++) {
            setTimeout(function(){
                // Gradually decrease the height of the message, such that the other messages go up
                let height = messages[i].offsetHeight;
                let anim = messages[i].animate([
                    { transform: "translateY(0px)", height: height + "px" },
                    { transform: "translateY(calc(-" + height + "px * 1.5))", height: "10px" }
                ], {
                    duration: 200,
                    easing: 'ease-in-out'
                })
                anim.onfinish = () => {
                    messages[i].remove();
                }
            }, 1000 * i);
        }
    }, MESSAGE_COOLDOWN_SECONDS * 1000);
});