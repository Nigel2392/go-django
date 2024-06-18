import { PageMenuController } from "./controllers/menu";
import { TippyController } from "./controllers/tippy";

window.Stimulus.register("pagemenu", PageMenuController);
window.Stimulus.register("tooltip", TippyController);