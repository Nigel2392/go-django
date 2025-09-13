class PagesRevisionCompareController extends window.StimulusController {
    static values = {
        url:      String,
        listUrl:  String,
    };
    
    declare readonly urlValue: string;
    declare readonly listUrlValue: string;
    declare chooser: typeof window.Chooser;
    
    connect() {
        this.element.addEventListener("click", function(e: Event) {
            e.preventDefault();

            const chooserConfig = {
                title: window.i18n.gettext("Select revision to compare"),
                listurl: this.listUrlValue,
                onChosen: (value: string, previewText: string, data?: any) => {
                    const newUrl = this.urlValue.replace("__ID__", value);
                    window.location.href = newUrl;
                }   
            }

            this.chooser = new window.Chooser(chooserConfig);
            this.chooser.open();
        }.bind(this));
    }
}

export { PagesRevisionCompareController };