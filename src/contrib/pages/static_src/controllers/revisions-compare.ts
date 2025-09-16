class PagesRevisionCompareController extends window.StimulusController {
    static values = {
        url:      String,
        listUrl:  String,
        currentId: Number,
    };
    
    declare readonly urlValue: string;
    declare readonly listUrlValue: string;
    declare readonly currentIdValue: string;
    declare chooser: typeof window.Chooser;
    
    connect() {
        this.element.addEventListener("click", function(e: Event) {
            e.preventDefault();

            const chooserConfig = {
                title: window.i18n.gettext("Select revision to compare"),
                listurl: this.listUrlValue,
                onChosen: (value: string, previewText: string, data?: any) => {
                    let valueInt = parseInt(value);
                    let idList = [this.currentIdValue, valueInt].sort((a, b) => a - b);
                    let newUrl = this.urlValue.
                        replace("__OLD_ID__", idList[0].toString()).
                        replace("__NEW_ID__", idList[1].toString());
                    window.location.href = newUrl;
                }   
            }

            this.chooser = new window.Chooser(chooserConfig);
            this.chooser.open();
        }.bind(this));
    }
}

export { PagesRevisionCompareController };