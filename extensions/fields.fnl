ns fields

# get value from map by giving path as string which has
# dot separated field strings
getv = func(srcmap path)
	get-next = func(nextmap left-parts)
		if( empty(left-parts)
			nextmap
			call(get-next get(nextmap head(left-parts)) rest(left-parts))
		)
	end

	call(get-next srcmap split(path '.'))
end

# get value from map by giving path as string of dot separated field names.
# return value is pair/list: <found> <value>
getlv = func(srcmap path)
	get-next = func(nextmap left-parts)
		if( empty(left-parts)
			list(true nextmap)
			call(func()
				found val = getl(nextmap head(left-parts)):
				if( found
					call(get-next val rest(left-parts))
					list(false val)
				)
			end)
		)
	end

	call(get-next srcmap split(path '.'))
end

# ------------------- tests -------------------------------------------------------------
test-1 = func()
	m = map(
		'person' map(
			'info' map(
				'age' 50
				'name' map(
					'firstname' 'Hank'
					'lastname'  'Hill'
				)
			)
		)
	)

	and(
		eq( call(getv m 'person.info.name') map('firstname' 'Hank', 'lastname' 'Hill') )
		eq( call(getv m 'person.info.name.firstname') 'Hank' )
		eq( call(getv m 'person.info.name.lastname') 'Hill' )
		eq( call(getv m 'person.info.age') 50 )
	)
end

test-2 = func()
	m = map(
		'person' map(
			'info' map(
				'age' 50
				'name' map(
					'firstname' 'Hank'
					'lastname'  'Hill'
				)
			)
		)
	)

	and(
		eq( call(getlv m 'person.info.name') list(true map('firstname' 'Hank', 'lastname' 'Hill')) )
		eq( call(getlv m 'person.info.name.firstname') list(true 'Hank') )
		eq( call(getlv m 'person.info.name.lastname') list(true 'Hill') )
		eq( call(getlv m 'person.info.age') list(true 50) )

		eq( list(false false) call(getlv m 'notfound.info.age') )
		eq( list(false false) call(getlv m '') )
	)
end

run-tests = proc()
	and(
		call(test-1)
		call(test-2)
	)
end

endns
