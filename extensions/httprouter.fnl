ns httprouter

new-router = proc(router)
	import stdhttp

	mux = call(stdhttp.mux)

	addr = get(router 'addr')
	routes = get(router 'routes')

	get-listen = func(listener)
		proc()
			import stdbytes

			middle-handler = proc(w r)
				uri-raw = get(r 'URI')
				uri = head(split(uri-raw '?'))
				pathl-1 = rest(split(uri '/'))
				pathl = if(eq(last(pathl-1) '') rrest(pathl-1) pathl-1)

				given-method = get(r 'method')
				method-found method-route = getl(routes given-method):

				handler-finder = proc(mroute)
					is-match = func(onepath)
						match-next = func(nrou npath rrou rpath retval)
							import stdstr

							next-val = cond(
								eq(nrou npath) true
								call(stdstr.startswith nrou ':') true
								false
							)

							if( or(empty(rpath) empty(rrou))
								next-val
								call(func()
									if( next-val
										call(match-next head(rrou) head(rpath) rest(rrou) rest(rpath) next-val)
										false
									)
								end)
							)
						end

						if( eq(len(onepath) len(pathl))
							call(match-next head(onepath) head(pathl) rest(onepath) rest(pathl) true)
							false
						)
					end

					import stdfu
					hlist = call(stdfu.filter mroute func(pair) call(is-match head(pair)) end)

					if( empty(hlist)
						list(false 'none' map())
						call(proc()
							pattern handler = head(hlist):
							parameters = call(read-parameters pattern pathl)
							list(true handler parameters)
						end)
					)
				end

				handler-caller = proc(mroute)
					is-handler-found handler params = call(handler-finder mroute):
					if( is-handler-found
						call(proc()
							rv = try(call(handler w r params) 'HTTP handler made RTE') # todo: plus additional info later
							if( eq('HTTP handler made RTE' rv)
								list(false 'server error' 500)
								list(true '' 200)
							)
						end)
						list(false sprintf('route not found (%v)' uri) 404)
					)
				end

				handler-ok err-text status-code = if( method-found
					call(handler-caller method-route)
					list(false sprintf('method not found (%s)(%v)' given-method uri) 404)
				):
				if( handler-ok
					'whatever'
					call(stdhttp.write-response w status-code call(stdbytes.str-to-bytes err-text))
				)
			end

			_ = call(stdhttp.reg-handler mux '/' middle-handler)
			call(listener)
		end
	end

	shutdown = proc()
		call(stdhttp.shutdown mux)
	end

	certf = if(in(router 'cert-file') get(router 'cert-file') '')
	keyf = if(in(router 'key-file') get(router 'key-file') '')

	# return services in map
	map(
		'listen'     call(get-listen proc() call(stdhttp.listen-and-serve mux addr) end)
		'listen-tls' call(get-listen proc() call(stdhttp.listen-and-serve-tls mux certf keyf addr) end)
		'shutdown'   shutdown
	)
end

read-parameters = func(pattern path)
	import stdstr

	iterate = func(pattern-left path-left params)
		if( empty(pattern-left)
			params
			call(func()
				next-in-pattern = head(pattern-left)
				next-in-path = head(path-left)
				next-params = if( call(stdstr.startswith next-in-pattern ':')
					put(params next-in-pattern next-in-path)
					params
				)
				call(iterate rest(pattern-left) rest(path-left) next-params)
			end)
		)
	end

	import stddbc
	_ = call(stddbc.assert eq(len(pattern) len(path)) 'not same length')

	call(iterate pattern path map())
end

test-it = proc()
	pattern = list('app' 'item' ':id' 'hmm' ':name')
	path = list('app' 'item' '123' 'hmm' 'myname')
	parameters = call(read-parameters pattern path)
	if( eq(parameters map(':id' '123' ':name' 'myname'))
		'ok'
		'failed'
	)
end

endns
