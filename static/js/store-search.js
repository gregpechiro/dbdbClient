
function query() {
    if (store != [] && store.length > 0) {
        var filter = new Function('doc', editor.getValue());
        result = store.filter(filter);
        page = 0;
        $('p#msgContent').text('Found ' + result.length + ' records');
        $('.navbar-center').addClass('hide');
        $('span#JSmsg').removeClass('hide');
        genResults(paginate(result));
    }

    if (result.length > 0) {
        var blob = setExport(result);
        $('a#queryExport').attr('href', window.URL.createObjectURL(blob));
        $('a#queryExport').attr('download', 'query_' + genFileName(storeName) + '.json');
        $('a#queryExport').removeAttr('disabled');
    }
}

$(document).ready(function() {
    editor.session.setMode("ace/mode/javascript");
    editor.renderer.setShowGutter(true);
    editor.setHighlightActiveLine(true);
    editor.setReadOnly(false);
    editor.setTheme("ace/theme/terminal");
    editor.setDisplayIndentGuides(true);
    editor.setFontSize(15);
    editor.getSession().on("changeAnnotation", function(){
        var annot = editor.getSession().getAnnotations();
        if (annot.length > 0) {
            $('button#search').attr('disabled', 'disabled');
            $('button#save-search').attr('disabled', 'disabled');
        } else {
            $('button#search').removeAttr('disabled');
            $('button#save-search').removeAttr('disabled');
        }
    });

    $('#save-search').click(function(e) {
        e.preventDefault();
        $('input#search').val(editor.getValue());
        $('form#save-search-form').submit();
    })

    $('#search').click(function() {
        query();
    });

    genResults(paginate(store));

    $('select#pageSize').change(function() {
        page = 0;
        pageSize =+ this.value;
        genResults(paginate(result));
    });

});
