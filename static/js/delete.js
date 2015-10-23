$(document).ready(function() {

	$('.delete-button').click(function() {
    	$('form#deleteForm').attr('action', $(this).attr('data-delete'));
    	$('label#message').html($(this).attr('data-message'));
		$('.navbar-center').addClass('hide');
    	$('span#delete-msg').removeClass('hide');
    });

    $('a#deleteCancel').click(function() {
    	$('form#deleteForm').attr('action', '');
		$('label#message').html('');
    	$('span#delete-msg').addClass('hide');
    });

});

function registerDelete() {
	$('.delete-button').click(function() {
    	$('form#deleteForm').attr('action', $(this).attr('data-delete'));
    	$('label#message').html($(this).attr('data-message'));
		$('.navbar-center').addClass('hide');
    	$('span#delete-msg').removeClass('hide');
    });
}
