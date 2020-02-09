import re

res = open('results.txt', "w")
res.write('replicas,nodes,remote,run,warehouses,tpmC\n')
for nodes in [3, 6, 12, 24, 48]:
    for remote in [1, 10, 100, 100000000]:
        for run in [1, 2, 3]:
            #file = '/Users/becca/go/src/github.com/cockroachdb/cockroach/artifacts-sav/tpccbench/nodes=%d/cpu=4/remote=%d/partition/run_%d/test.log' % (nodes, remote, run)
            file = '/Users/becca/go/src/github.com/cockroachdb/cockroach/artifacts/tpccbench/nodes=%d/cpu=4/remote=%d/partition/run_%d/test.log' % (nodes, remote, run)
            print file
            maxtpmC = 0
            maxW = 0
            try:
                with open(file, "r") as f: 
                    for line in f:
                        t = re.match('.*PASS: tpcc \d+ resulted in (\d+.?\d+) tpmC.*', line)
                        if t:
                            if float(t.groups()[0]) > maxtpmC:
                                maxtpmC = float(t.groups()[0])

                        w = re.match('MAX WAREHOUSES = (\d+)', line)
                        if w:
                            maxW = int(w.groups()[0])

                #res.write('3,%d,%d,%d,%d,%f\n' % (nodes, int(100.0/remote), run, maxW, maxtpmC))
                res.write('5,%d,%d,%d,%d,%f\n' % (nodes, int(100.0/remote), run, maxW, maxtpmC))
            except IOError:
                continue

res.close()
