
(()=>{"use strict";var t={523:(t,e,n)=>{n.d(e,{A:()=>o});var s=n(601),i=n.n(s),r=n(314),a=n.n(r)()(i());a.push([t.id,".ce-tune-alignment--right {\n    text-align: right;\n}\n.ce-tune-alignment--center {\n    text-align: center;\n}\n.ce-tune-alignment--left {\n    text-align: left;\n}",""]);const o=a},314:t=>{t.exports=function(t){var e=[];return e.toString=function(){return this.map((function(e){var n="",s=void 0!==e[5];return e[4]&&(n+="@supports (".concat(e[4],") {")),e[2]&&(n+="@media ".concat(e[2]," {")),s&&(n+="@layer".concat(e[5].length>0?" ".concat(e[5]):""," {")),n+=t(e),s&&(n+="}"),e[2]&&(n+="}"),e[4]&&(n+="}"),n})).join("")},e.i=function(t,n,s,i,r){"string"==typeof t&&(t=[[null,t,void 0]]);var a={};if(s)for(var o=0;o<this.length;o++){var c=this[o][0];null!=c&&(a[c]=!0)}for(var l=0;l<t.length;l++){var h=[].concat(t[l]);s&&a[h[0]]||(void 0!==r&&(void 0===h[5]||(h[1]="@layer".concat(h[5].length>0?" ".concat(h[5]):""," {").concat(h[1],"}")),h[5]=r),n&&(h[2]?(h[1]="@media ".concat(h[2]," {").concat(h[1],"}"),h[2]=n):h[2]=n),i&&(h[4]?(h[1]="@supports (".concat(h[4],") {").concat(h[1],"}"),h[4]=i):h[4]="".concat(i)),e.push(h))}},e}},601:t=>{t.exports=function(t){return t[1]}},72:t=>{var e=[];function n(t){for(var n=-1,s=0;s<e.length;s++)if(e[s].identifier===t){n=s;break}return n}function s(t,s){for(var r={},a=[],o=0;o<t.length;o++){var c=t[o],l=s.base?c[0]+s.base:c[0],h=r[l]||0,u="".concat(l," ").concat(h);r[l]=h+1;var d=n(u),p={css:c[1],media:c[2],sourceMap:c[3],supports:c[4],layer:c[5]};if(-1!==d)e[d].references++,e[d].updater(p);else{var g=i(p,s);s.byIndex=o,e.splice(o,0,{identifier:u,updater:g,references:1})}a.push(u)}return a}function i(t,e){var n=e.domAPI(e);return n.update(t),function(e){if(e){if(e.css===t.css&&e.media===t.media&&e.sourceMap===t.sourceMap&&e.supports===t.supports&&e.layer===t.layer)return;n.update(t=e)}else n.remove()}}t.exports=function(t,i){var r=s(t=t||[],i=i||{});return function(t){t=t||[];for(var a=0;a<r.length;a++){var o=n(r[a]);e[o].references--}for(var c=s(t,i),l=0;l<r.length;l++){var h=n(r[l]);0===e[h].references&&(e[h].updater(),e.splice(h,1))}r=c}}},659:t=>{var e={};t.exports=function(t,n){var s=function(t){if(void 0===e[t]){var n=document.querySelector(t);if(window.HTMLIFrameElement&&n instanceof window.HTMLIFrameElement)try{n=n.contentDocument.head}catch(t){n=null}e[t]=n}return e[t]}(t);if(!s)throw new Error("Couldn't find a style target. This probably means that the value for the 'insert' parameter is invalid.");s.appendChild(n)}},540:t=>{t.exports=function(t){var e=document.createElement("style");return t.setAttributes(e,t.attributes),t.insert(e,t.options),e}},56:(t,e,n)=>{t.exports=function(t){var e=n.nc;e&&t.setAttribute("nonce",e)}},825:t=>{t.exports=function(t){if("undefined"==typeof document)return{update:function(){},remove:function(){}};var e=t.insertStyleElement(t);return{update:function(n){!function(t,e,n){var s="";n.supports&&(s+="@supports (".concat(n.supports,") {")),n.media&&(s+="@media ".concat(n.media," {"));var i=void 0!==n.layer;i&&(s+="@layer".concat(n.layer.length>0?" ".concat(n.layer):""," {")),s+=n.css,i&&(s+="}"),n.media&&(s+="}"),n.supports&&(s+="}");var r=n.sourceMap;r&&"undefined"!=typeof btoa&&(s+="\n/*# sourceMappingURL=data:application/json;base64,".concat(btoa(unescape(encodeURIComponent(JSON.stringify(r))))," */")),e.styleTagTransform(s,t,e.options)}(e,t,n)},remove:function(){!function(t){if(null===t.parentNode)return!1;t.parentNode.removeChild(t)}(e)}}}},113:t=>{t.exports=function(t,e){if(e.styleSheet)e.styleSheet.cssText=t;else{for(;e.firstChild;)e.removeChild(e.firstChild);e.appendChild(document.createTextNode(t))}}}},e={};function n(s){var i=e[s];if(void 0!==i)return i.exports;var r=e[s]={id:s,exports:{}};return t[s](r,r.exports,n),r.exports}n.n=t=>{var e=t&&t.__esModule?()=>t.default:()=>t;return n.d(e,{a:e}),e},n.d=(t,e)=>{for(var s in e)n.o(e,s)&&!n.o(t,s)&&Object.defineProperty(t,s,{enumerable:!0,get:e[s]})},n.o=(t,e)=>Object.prototype.hasOwnProperty.call(t,e),n.nc=void 0,(()=>{var t=n(72),e=n.n(t),s=n(825),i=n.n(s),r=n(659),a=n.n(r),o=n(56),c=n.n(o),l=n(540),h=n.n(l),u=n(113),d=n.n(u),p=n(523),g={};function m(t,e="",n={}){const s=document.createElement(t);Array.isArray(e)?s.classList.add(...e):e&&s.classList.add(e);for(const t in n)s[t]=n[t];return s}g.styleTagTransform=d(),g.setAttributes=c(),g.insert=a().bind(null,"head"),g.domAPI=i(),g.insertStyleElement=h(),e()(p.A,g),p.A&&p.A.locals&&p.A.locals;class f{constructor({api:t,data:e,config:n,block:s}){this.api=t,this.block=s,this.settings=n,this.data=e||{alignment:this.getAlignment()},this.alignmentSettings=[{name:"left",icon:'<svg xmlns="http://www.w3.org/2000/svg" id="Layer" enable-background="new 0 0 64 64" height="20" viewBox="0 0 64 64" width="20"><path d="m54 8h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 52h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m10 23h28c1.104 0 2-.896 2-2s-.896-2-2-2h-28c-1.104 0-2 .896-2 2s.896 2 2 2z"/><path d="m54 30h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m10 45h28c1.104 0 2-.896 2-2s-.896-2-2-2h-28c-1.104 0-2 .896-2 2s.896 2 2 2z"/></svg>'},{name:"center",icon:'<svg xmlns="http://www.w3.org/2000/svg" id="Layer" enable-background="new 0 0 64 64" height="20" viewBox="0 0 64 64" width="20"><path d="m54 8h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 52h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m46 23c1.104 0 2-.896 2-2s-.896-2-2-2h-28c-1.104 0-2 .896-2 2s.896 2 2 2z"/><path d="m54 30h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m46 45c1.104 0 2-.896 2-2s-.896-2-2-2h-28c-1.104 0-2 .896-2 2s.896 2 2 2z"/></svg>'},{name:"right",icon:'<svg xmlns="http://www.w3.org/2000/svg" id="Layer" enable-background="new 0 0 64 64" height="20" viewBox="0 0 64 64" width="20"><path d="m54 8h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 52h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 19h-28c-1.104 0-2 .896-2 2s.896 2 2 2h28c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 30h-44c-1.104 0-2 .896-2 2s.896 2 2 2h44c1.104 0 2-.896 2-2s-.896-2-2-2z"/><path d="m54 41h-28c-1.104 0-2 .896-2 2s.896 2 2 2h28c1.104 0 2-.896 2-2s-.896-2-2-2z"/></svg>'}],this._CSS={alignment:{left:"ce-tune-alignment--left",center:"ce-tune-alignment--center",right:"ce-tune-alignment--right"}}}static get DEFAULT_ALIGNMENT(){return"left"}static get isTune(){return!0}getAlignment(){var t,e;if(this.constructor.blockAlignment){let t=this.constructor.blockAlignment;return this.constructor.blockAlignment=null,t}return(null===(t=this.settings)||void 0===t?void 0:t.blocks)&&this.settings.blocks.hasOwnProperty(this.block.name)?this.settings.blocks[this.block.name]:(null===(e=this.settings)||void 0===e?void 0:e.default)?this.settings.default:f.DEFAULT_ALIGNMENT}wrap(t){return this.wrapper=m("div"),this.wrapper.classList.toggle(this._CSS.alignment[this.data.alignment]),this.wrapper.append(t),setTimeout((()=>{this.api.listeners.on(this.block.holder,"keydown",(t=>{"Enter"===t.key&&(this.constructor.blockAlignment=this.data.alignment)}))}),0),this.wrapper}render(){const t=m("div");return this.alignmentSettings.map((e=>{const n=document.createElement("button");return n.classList.add(this.api.styles.settingsButton),n.innerHTML=e.icon,n.type="button",n.classList.toggle(this.api.styles.settingsButtonActive,e.name===this.data.alignment),this.api.tooltip.onHover(n,this.api.i18n.t(`Align ${e.name}`),{placement:"top",offset:5}),t.appendChild(n),n})).forEach(((t,e,n)=>{t.addEventListener("click",(()=>{const t=this.alignmentSettings[e].name;this.data={alignment:t},n.forEach(((t,e)=>{const{name:n}=this.alignmentSettings[e];t.classList.toggle(this.api.styles.settingsButtonActive,n===this.data.alignment),this.wrapper.classList.toggle(this._CSS.alignment[n],n===this.data.alignment)})),this.block.dispatchChange()}))})),t}save(){return this.data}}window.AlignmentBlockTune=f})()})();
