/*!
 ColReorder 1.5.1
 ©2010-2018 SpryMedia Ltd - datatables.net/license
*/
(function(e){"function"===typeof define&&define.amd?define(["jquery","datatables.net"],function(o){return e(o,window,document)}):"object"===typeof exports?module.exports=function(o,l){o||(o=window);if(!l||!l.fn.dataTable)l=require("datatables.net")(o,l).$;return e(l,o,o.document)}:e(jQuery,window,document)})(function(e,o,l,r){function q(a){for(var b=[],c=0,f=a.length;c<f;c++)b[a[c]]=c;return b}function p(a,b,c){b=a.splice(b,1)[0];a.splice(c,0,b)}function s(a,b,c){for(var f=[],e=0,d=a.childNodes.length;e<
d;e++)1==a.childNodes[e].nodeType&&f.push(a.childNodes[e]);b=f[b];null!==c?a.insertBefore(b,f[c]):a.appendChild(b)}var t=e.fn.dataTable;e.fn.dataTableExt.oApi.fnColReorder=function(a,b,c,f,g){var d,h,j,m,i,l=a.aoColumns.length,k;i=function(a,b,c){if(a[b]&&"function"!==typeof a[b]){var d=a[b].split("."),f=d.shift();isNaN(1*f)||(a[b]=c[1*f]+"."+d.join("."))}};if(b!=c)if(0>b||b>=l)this.oApi._fnLog(a,1,"ColReorder 'from' index is out of bounds: "+b);else if(0>c||c>=l)this.oApi._fnLog(a,1,"ColReorder 'to' index is out of bounds: "+
	c);else{j=[];d=0;for(h=l;d<h;d++)j[d]=d;p(j,b,c);var n=q(j);d=0;for(h=a.aaSorting.length;d<h;d++)a.aaSorting[d][0]=n[a.aaSorting[d][0]];if(null!==a.aaSortingFixed){d=0;for(h=a.aaSortingFixed.length;d<h;d++)a.aaSortingFixed[d][0]=n[a.aaSortingFixed[d][0]]}d=0;for(h=l;d<h;d++){k=a.aoColumns[d];j=0;for(m=k.aDataSort.length;j<m;j++)k.aDataSort[j]=n[k.aDataSort[j]];k.idx=n[k.idx]}e.each(a.aLastSort,function(b,c){a.aLastSort[b].src=n[c.src]});d=0;for(h=l;d<h;d++)k=a.aoColumns[d],"number"==typeof k.mData?
	k.mData=n[k.mData]:e.isPlainObject(k.mData)&&(i(k.mData,"_",n),i(k.mData,"filter",n),i(k.mData,"sort",n),i(k.mData,"type",n));if(a.aoColumns[b].bVisible){i=this.oApi._fnColumnIndexToVisible(a,b);m=null;for(d=c<b?c:c+1;null===m&&d<l;)m=this.oApi._fnColumnIndexToVisible(a,d),d++;j=a.nTHead.getElementsByTagName("tr");d=0;for(h=j.length;d<h;d++)s(j[d],i,m);if(null!==a.nTFoot){j=a.nTFoot.getElementsByTagName("tr");d=0;for(h=j.length;d<h;d++)s(j[d],i,m)}d=0;for(h=a.aoData.length;d<h;d++)null!==a.aoData[d].nTr&&
s(a.aoData[d].nTr,i,m)}p(a.aoColumns,b,c);d=0;for(h=l;d<h;d++)a.oApi._fnColumnOptions(a,d,{});p(a.aoPreSearchCols,b,c);d=0;for(h=a.aoData.length;d<h;d++){m=a.aoData[d];if(k=m.anCells){p(k,b,c);j=0;for(i=k.length;j<i;j++)k[j]&&k[j]._DT_CellIndex&&(k[j]._DT_CellIndex.column=j)}"dom"!==m.src&&e.isArray(m._aData)&&p(m._aData,b,c)}d=0;for(h=a.aoHeader.length;d<h;d++)p(a.aoHeader[d],b,c);if(null!==a.aoFooter){d=0;for(h=a.aoFooter.length;d<h;d++)p(a.aoFooter[d],b,c)}(g||g===r)&&e.fn.dataTable.Api(a).rows().invalidate();
	d=0;for(h=l;d<h;d++)e(a.aoColumns[d].nTh).off(".DT"),this.oApi._fnSortAttachListener(a,a.aoColumns[d].nTh,d);e(a.oInstance).trigger("column-reorder.dt",[a,{from:b,to:c,mapping:n,drop:f,iFrom:b,iTo:c,aiInvertMapping:n}])}};var i=function(a,b){var c=(new e.fn.dataTable.Api(a)).settings()[0];if(c._colReorder)return c._colReorder;!0===b&&(b={});var f=e.fn.dataTable.camelToHungarian;f&&(f(i.defaults,i.defaults,!0),f(i.defaults,b||{}));this.s={dt:null,enable:null,init:e.extend(!0,{},i.defaults,b),fixed:0,
	fixedRight:0,reorderCallback:null,mouse:{startX:-1,startY:-1,offsetX:-1,offsetY:-1,target:-1,targetIndex:-1,fromIndex:-1},aoTargets:[]};this.dom={drag:null,pointer:null};this.s.enable=this.s.init.bEnable;this.s.dt=c;this.s.dt._colReorder=this;this._fnConstruct();return this};e.extend(i.prototype,{fnEnable:function(a){if(!1===a)return fnDisable();this.s.enable=!0},fnDisable:function(){this.s.enable=!1},fnReset:function(){this._fnOrderColumns(this.fnOrder());return this},fnGetCurrentOrder:function(){return this.fnOrder()},
	fnOrder:function(a,b){var c=[],f,g,d=this.s.dt.aoColumns;if(a===r){f=0;for(g=d.length;f<g;f++)c.push(d[f]._ColReorder_iOrigCol);return c}if(b){d=this.fnOrder();f=0;for(g=a.length;f<g;f++)c.push(e.inArray(a[f],d));a=c}this._fnOrderColumns(q(a));return this},fnTranspose:function(a,b){b||(b="toCurrent");var c=this.fnOrder(),f=this.s.dt.aoColumns;return"toCurrent"===b?!e.isArray(a)?e.inArray(a,c):e.map(a,function(a){return e.inArray(a,c)}):!e.isArray(a)?f[a]._ColReorder_iOrigCol:e.map(a,function(a){return f[a]._ColReorder_iOrigCol})},
	_fnConstruct:function(){var a=this,b=this.s.dt.aoColumns.length,c=this.s.dt.nTable,f;this.s.init.iFixedColumns&&(this.s.fixed=this.s.init.iFixedColumns);this.s.init.iFixedColumnsLeft&&(this.s.fixed=this.s.init.iFixedColumnsLeft);this.s.fixedRight=this.s.init.iFixedColumnsRight?this.s.init.iFixedColumnsRight:0;this.s.init.fnReorderCallback&&(this.s.reorderCallback=this.s.init.fnReorderCallback);for(f=0;f<b;f++)f>this.s.fixed-1&&f<b-this.s.fixedRight&&this._fnMouseListener(f,this.s.dt.aoColumns[f].nTh),
		this.s.dt.aoColumns[f]._ColReorder_iOrigCol=f;this.s.dt.oApi._fnCallbackReg(this.s.dt,"aoStateSaveParams",function(b,c){a._fnStateSave.call(a,c)},"ColReorder_State");var g=null;this.s.init.aiOrder&&(g=this.s.init.aiOrder.slice());this.s.dt.oLoadedState&&("undefined"!=typeof this.s.dt.oLoadedState.ColReorder&&this.s.dt.oLoadedState.ColReorder.length==this.s.dt.aoColumns.length)&&(g=this.s.dt.oLoadedState.ColReorder);if(g)if(a.s.dt._bInitComplete)b=q(g),a._fnOrderColumns.call(a,b);else{var d=!1;e(c).on("draw.dt.colReorder",
		function(){if(!a.s.dt._bInitComplete&&!d){d=true;var b=q(g);a._fnOrderColumns.call(a,b)}})}else this._fnSetColumnIndexes();e(c).on("destroy.dt.colReorder",function(){e(c).off("destroy.dt.colReorder draw.dt.colReorder");e.each(a.s.dt.aoColumns,function(a,b){e(b.nTh).off(".ColReorder");e(b.nTh).removeAttr("data-column-index")});a.s.dt._colReorder=null;a.s=null})},_fnOrderColumns:function(a){var b=!1;if(a.length!=this.s.dt.aoColumns.length)this.s.dt.oInstance.oApi._fnLog(this.s.dt,1,"ColReorder - array reorder does not match known number of columns. Skipping.");
	else{for(var c=0,f=a.length;c<f;c++){var g=e.inArray(c,a);c!=g&&(p(a,g,c),this.s.dt.oInstance.fnColReorder(g,c,!0,!1),b=!0)}this._fnSetColumnIndexes();b&&(e.fn.dataTable.Api(this.s.dt).rows().invalidate(),(""!==this.s.dt.oScroll.sX||""!==this.s.dt.oScroll.sY)&&this.s.dt.oInstance.fnAdjustColumnSizing(!1),this.s.dt.oInstance.oApi._fnSaveState(this.s.dt),null!==this.s.reorderCallback&&this.s.reorderCallback.call(this))}},_fnStateSave:function(a){var b,c,f,g=this.s.dt.aoColumns;a.ColReorder=[];if(a.aaSorting){for(b=
		                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    0;b<a.aaSorting.length;b++)a.aaSorting[b][0]=g[a.aaSorting[b][0]]._ColReorder_iOrigCol;var d=e.extend(!0,[],a.aoSearchCols);b=0;for(c=g.length;b<c;b++)f=g[b]._ColReorder_iOrigCol,a.aoSearchCols[f]=d[b],a.abVisCols[f]=g[b].bVisible,a.ColReorder.push(f)}else if(a.order){for(b=0;b<a.order.length;b++)a.order[b][0]=g[a.order[b][0]]._ColReorder_iOrigCol;d=e.extend(!0,[],a.columns);b=0;for(c=g.length;b<c;b++)f=g[b]._ColReorder_iOrigCol,a.columns[f]=d[b],a.ColReorder.push(f)}},_fnMouseListener:function(a,
	                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            b){var c=this;e(b).on("mousedown.ColReorder",function(a){c.s.enable&&c._fnMouseDown.call(c,a,b)}).on("touchstart.ColReorder",function(a){c.s.enable&&c._fnMouseDown.call(c,a,b)})},_fnMouseDown:function(a,b){var c=this,f=e(a.target).closest("th, td").offset(),g=parseInt(e(b).attr("data-column-index"),10);g!==r&&(this.s.mouse.startX=this._fnCursorPosition(a,"pageX"),this.s.mouse.startY=this._fnCursorPosition(a,"pageY"),this.s.mouse.offsetX=this._fnCursorPosition(a,"pageX")-f.left,this.s.mouse.offsetY=
		this._fnCursorPosition(a,"pageY")-f.top,this.s.mouse.target=this.s.dt.aoColumns[g].nTh,this.s.mouse.targetIndex=g,this.s.mouse.fromIndex=g,this._fnRegions(),e(l).on("mousemove.ColReorder touchmove.ColReorder",function(a){c._fnMouseMove.call(c,a)}).on("mouseup.ColReorder touchend.ColReorder",function(a){c._fnMouseUp.call(c,a)}))},_fnMouseMove:function(a){if(null===this.dom.drag){if(5>Math.pow(Math.pow(this._fnCursorPosition(a,"pageX")-this.s.mouse.startX,2)+Math.pow(this._fnCursorPosition(a,"pageY")-
		this.s.mouse.startY,2),0.5))return;this._fnCreateDragNode()}this.dom.drag.css({left:this._fnCursorPosition(a,"pageX")-this.s.mouse.offsetX,top:this._fnCursorPosition(a,"pageY")-this.s.mouse.offsetY});for(var b=!1,c=this.s.mouse.toIndex,f=1,e=this.s.aoTargets.length;f<e;f++)if(this._fnCursorPosition(a,"pageX")<this.s.aoTargets[f-1].x+(this.s.aoTargets[f].x-this.s.aoTargets[f-1].x)/2){this.dom.pointer.css("left",this.s.aoTargets[f-1].x);this.s.mouse.toIndex=this.s.aoTargets[f-1].to;b=!0;break}b||(this.dom.pointer.css("left",
		this.s.aoTargets[this.s.aoTargets.length-1].x),this.s.mouse.toIndex=this.s.aoTargets[this.s.aoTargets.length-1].to);this.s.init.bRealtime&&c!==this.s.mouse.toIndex&&(this.s.dt.oInstance.fnColReorder(this.s.mouse.fromIndex,this.s.mouse.toIndex),this.s.mouse.fromIndex=this.s.mouse.toIndex,(""!==this.s.dt.oScroll.sX||""!==this.s.dt.oScroll.sY)&&this.s.dt.oInstance.fnAdjustColumnSizing(!1),this._fnRegions())},_fnMouseUp:function(){e(l).off(".ColReorder");null!==this.dom.drag&&(this.dom.drag.remove(),
		this.dom.pointer.remove(),this.dom.drag=null,this.dom.pointer=null,this.s.dt.oInstance.fnColReorder(this.s.mouse.fromIndex,this.s.mouse.toIndex,!0),this._fnSetColumnIndexes(),(""!==this.s.dt.oScroll.sX||""!==this.s.dt.oScroll.sY)&&this.s.dt.oInstance.fnAdjustColumnSizing(!1),this.s.dt.oInstance.oApi._fnSaveState(this.s.dt),null!==this.s.reorderCallback&&this.s.reorderCallback.call(this))},_fnRegions:function(){var a=this.s.dt.aoColumns;this.s.aoTargets.splice(0,this.s.aoTargets.length);this.s.aoTargets.push({x:e(this.s.dt.nTable).offset().left,
		to:0});for(var b=0,c=this.s.aoTargets[0].x,f=0,g=a.length;f<g;f++)f!=this.s.mouse.fromIndex&&b++,a[f].bVisible&&"none"!==a[f].nTh.style.display&&(c+=e(a[f].nTh).outerWidth(),this.s.aoTargets.push({x:c,to:b}));0!==this.s.fixedRight&&this.s.aoTargets.splice(this.s.aoTargets.length-this.s.fixedRight);0!==this.s.fixed&&this.s.aoTargets.splice(0,this.s.fixed)},_fnCreateDragNode:function(){var a=""!==this.s.dt.oScroll.sX||""!==this.s.dt.oScroll.sY,b=this.s.dt.aoColumns[this.s.mouse.targetIndex].nTh,c=b.parentNode,
		f=c.parentNode,g=f.parentNode,d=e(b).clone();this.dom.drag=e(g.cloneNode(!1)).addClass("DTCR_clonedTable").append(e(f.cloneNode(!1)).append(e(c.cloneNode(!1)).append(d[0]))).css({position:"absolute",top:0,left:0,width:e(b).outerWidth(),height:e(b).outerHeight()}).appendTo("body");this.dom.pointer=e("<div></div>").addClass("DTCR_pointer").css({position:"absolute",top:a?e("div.dataTables_scroll",this.s.dt.nTableWrapper).offset().top:e(this.s.dt.nTable).offset().top,height:a?e("div.dataTables_scroll",
			this.s.dt.nTableWrapper).height():e(this.s.dt.nTable).height()}).appendTo("body")},_fnSetColumnIndexes:function(){e.each(this.s.dt.aoColumns,function(a,b){e(b.nTh).attr("data-column-index",a)})},_fnCursorPosition:function(a,b){return-1!==a.type.indexOf("touch")?a.originalEvent.touches[0][b]:a[b]}});i.defaults={aiOrder:null,bEnable:!0,bRealtime:!0,iFixedColumnsLeft:0,iFixedColumnsRight:0,fnReorderCallback:null};i.version="1.5.1";e.fn.dataTable.ColReorder=i;e.fn.DataTable.ColReorder=i;"function"==typeof e.fn.dataTable&&
"function"==typeof e.fn.dataTableExt.fnVersionCheck&&e.fn.dataTableExt.fnVersionCheck("1.10.8")?e.fn.dataTableExt.aoFeatures.push({fnInit:function(a){var b=a.oInstance;a._colReorder?b.oApi._fnLog(a,1,"ColReorder attempted to initialise twice. Ignoring second"):(b=a.oInit,new i(a,b.colReorder||b.oColReorder||{}));return null},cFeature:"R",sFeature:"ColReorder"}):alert("Warning: ColReorder requires DataTables 1.10.8 or greater - www.datatables.net/download");e(l).on("preInit.dt.colReorder",function(a,
                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      b){if("dt"===a.namespace){var c=b.oInit.colReorder,f=t.defaults.colReorder;if(c||f)f=e.extend({},c,f),!1!==c&&new i(b,f)}});e.fn.dataTable.Api.register("colReorder.reset()",function(){return this.iterator("table",function(a){a._colReorder.fnReset()})});e.fn.dataTable.Api.register("colReorder.order()",function(a,b){return a?this.iterator("table",function(c){c._colReorder.fnOrder(a,b)}):this.context.length?this.context[0]._colReorder.fnOrder():null});e.fn.dataTable.Api.register("colReorder.transpose()",
	function(a,b){return this.context.length&&this.context[0]._colReorder?this.context[0]._colReorder.fnTranspose(a,b):a});e.fn.dataTable.Api.register("colReorder.move()",function(a,b,c,e){this.context.length&&this.context[0]._colReorder.s.dt.oInstance.fnColReorder(a,b,c,e);return this});e.fn.dataTable.Api.register("colReorder.enable()",function(a){return this.iterator("table",function(b){b._colReorder&&b._colReorder.fnEnable(a)})});e.fn.dataTable.Api.register("colReorder.disable()",function(){return this.iterator("table",
	function(a){a._colReorder&&a._colReorder.fnDisable()})});return i});