var pageSize = 10;
var page = 0;
var pages;
var test;
//var lb = 0;
//var ub = 9;

function paginate(dataSet) {
    pages = Math.ceil(dataSet.length / pageSize);
    var lb;
    var ub;

    //lb = (pages < 11 || page < 6) ? 0 : page - 5;
    ub = ((pages - page) >= 5) ? page + 4 : pages - 1 ;

    if (page < 6) {
		ub = (pages > 9) ? 9 : pages - 1
	}
    lb = (((ub - 9) > 0) ? ub - 9 : 0);
    /*if (pages < 11) {
        lb = 0;
        ub = pages;
    } else if (page > 5) {
        lb = page - 5;
        ub = lb + 9;
    }  else {
        lb = 0;
        ub = 9;
    }*/
    generatePaginator(lb, ub, pages);
    var beg = page * pageSize;
    var end = ((page * pageSize) + pageSize);
    return dataSet.slice(beg, end);
}

function generatePaginator(lb, ub, pages) {
    var paginator = $('#paginator');
    paginator.html('');
    var prev = $('<li id="prev"><a href="#" aria-label="Previous"><span aria-hidden="true">&laquo;</span</a></li>');
    var next = $('<li id="next"><a href="#" aria-label="Next"><span aria-hidden="true">&raquo;</span></a></li>');
    if (page == 0) {
        prev.addClass('disabled');
    }
    if (page === (pages - 1)) {
        next.addClass('disabled');
    }
    prev.click(function() {
        if (page > 0) {
            page--;
            genResults(paginate(result));
        }
    });
    next.click(function() {
        if (page + 1 < pages) {
            page++;
            genResults(paginate(result));
        }
    });
    paginator.append(prev);
    for (var i =  lb; i <= ub; i++) {
        var elem = $('<li data-page="' + i + '"><a href="#">' + (i+1) + '</a></li>');
        if (page == i) {
            elem.addClass('active');
        }
        elem.click(function() {
            page =+ this.getAttribute('data-page');
            genResults(paginate(result));
        });
        paginator.append(elem);
    }
    paginator.append(next);
}

function genResults(results) {
    var i;
    group = $('<div class="panel-group" id="accordion" role="tablist" aria-multiselectable="true"></div>');
    for (i = 0; i < results.length; i++) {
        doc = $('<div class="panel panel-default">' +
            '<div class="panel-heading clearfix" role="tab" id="headingOne">' +
                '<h4 class="panel-title">' +
                    '<a role="button" class="btn btn-success btn-xs" style="color: #fff;" data-toggle="collapse" data-parent="#accordion" href="#collapse' + results[i].id + '" aria-expanded="true" aria-controls="collapse' + results[i].id + '">' +
                        'ID: ' + results[i].id +
                    '</a>' +
                    '<span class="pull-right">' +
                        '<a href="/' + storeName + '/' + results[i].id + '">Edit</a>' +
                        '&nbsp;&nbsp;&nbsp;&nbsp;' +
                        '<a href="#" data-message="Are you sure you would like to delete this record?" data-delete="/' + storeName + '/' + results[i].id + '/del" class="delete-button text-danger">Delete</a>' +
                    '</span>' +
                '</h4>' +
            '</div>' +
            '<div id="collapse' + results[i].id + '" class="panel-collapse collapse" role="tabpanel" aria-labelledby="heading' + results[i].id + '">' +
                '<div class="panel-body">' +
                    '<pre id="editor' + results[i].id + '" style="height:400px;">' + JSON.stringify(results[i].data, null, 4) + '</pre>' +
                '</div>' +
            '</div>' +
        '</div>')
        group.append(doc)
    }
    $('#results').html(group)
    registerDelete();
}
