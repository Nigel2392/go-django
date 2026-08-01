import { PageMenuController } from "./controllers/menu";
import { PagesRevisionCompareController } from "./controllers/revisions-compare";

window.Stimulus.register("pagemenu", PageMenuController);
window.Stimulus.register("pages-revision-compare", PagesRevisionCompareController);
