#!/bin/bash

hosts=("becca-test-0001" "becca-test-0002" "becca-test-0003")
var=0

while true
do
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics stock1 on s_i_id from tpcc.stock;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics stock2 on s_w_id from tpcc.stock;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics customer1 on c_id from tpcc.customer;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics customer2 on c_w_id from tpcc.customer;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics customer3 on c_d_id from tpcc.customer;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics customer4 on c_last from tpcc.customer;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics customer5 on c_first from tpcc.customer;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics district1 on d_id from tpcc.district;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics district2 on d_w_id from tpcc.district;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics history1 on h_w_id from tpcc.history;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics history2 on h_d_id from tpcc.history;" 
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics history3 on h_c_w_id from tpcc.history;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics history4 on h_c_d_id from tpcc.history;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics history5 on h_c_id from tpcc.history;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics item1 on i_id from tpcc.item;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics new_order1 on no_w_id from tpcc.new_order;" 
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics new_order2 on no_d_id from tpcc.new_order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics new_order3 on no_o_id from tpcc.new_order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order1 on o_w_id from tpcc.order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order2 on o_d_id from tpcc.order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order3 on o_id from tpcc.order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order4 on o_c_id from tpcc.order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order5 on o_carrier_id from tpcc.order;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line1 on ol_w_id from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line2 on ol_d_id from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line3 on ol_o_id from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line4 on ol_number from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line5 on ol_i_id from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics order_line6 on ol_supply_w_id from tpcc.order_line;"
    ./cockroach sql --insecure --host ${hosts[$((var=(var+1)%3))]} --execute="create statistics warehouse1 on w_id from tpcc.warehouse;"
done
